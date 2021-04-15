// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/Orkestra/pkg"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/Azure/Orkestra/pkg/workflow"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	helmopv1 "github.com/fluxcd/helm-operator/pkg/apis/helm.fluxcd.io/v1"
	"github.com/go-logr/logr"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
)

const (
	appgroupNameKey                   = "appgroup"
	finalizer                         = "application-group-finalizer"
	lastSuccessfulApplicationGroupKey = "orkestra/last-successful-applicationgroup"
)

var (
	ErrWorkflowInFailureStatus    = errors.New("workflow in failure status")
	ErrHelmReleaseInFailureStatus = errors.New("helmrelease in failure status")
)

// ApplicationGroupReconciler reconciles a ApplicationGroup object
type ApplicationGroupReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	Engine workflow.Engine

	// RegistryClient interacts with the helm registries to pull and push charts
	RegistryClient *registry.Client

	// WorkflowNS is the namespace to which (generated) Argo Workflow object is deployed
	WorkflowNS string

	// StagingRepoName is the nickname for the repository used for staging artifacts before being deployed using the HelmRelease object
	StagingRepoName string

	// TargetDir to stage the charts before pushing
	TargetDir string

	// Recorder generates kubernetes events
	Recorder record.EventRecorder

	// DisableRemediation for debugging purposes
	// The object and associated Workflow, HelmReleases will
	// not be cleaned up
	DisableRemediation bool

	// CleanupDownloadedCharts signals the controller to delete the
	// fetched charts after they have been repackaged and pushed to staging
	CleanupDownloadedCharts bool

	// lastSuccessfulApplicationGroup holds the applicationgroup spec body from the last
	// successful reconciliation of the ApplicationGroup. This is set after every successful
	// reconciliation.
	lastSuccessfulApplicationGroup *orkestrav1alpha1.ApplicationGroup
}

// +kubebuilder:rbac:groups=orkestra.azure.microsoft.com,resources=applicationgroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=orkestra.azure.microsoft.com,resources=applicationgroups/status,verbs=get;update;patch

func (r *ApplicationGroupReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var requeue bool
	var err error
	var appGroup orkestrav1alpha1.ApplicationGroup

	ctx := context.Background()
	logr := r.Log.WithValues(appgroupNameKey, req.NamespacedName.Name)

	if err := r.Get(ctx, req.NamespacedName, &appGroup); err != nil {
		if kerrors.IsNotFound(err) {
			logr.V(3).Info("skip reconciliation since AppGroup instance not found on the cluster")
			return ctrl.Result{}, nil
		}
		logr.Error(err, "unable to fetch ApplicationGroup instance")
		return ctrl.Result{}, err
	}

	// Check if this is an update event to the ApplicationGroup
	// in which case unmarshal the last successful spec into a
	// variable
	if appGroup.GetAnnotations() != nil {
		last := &orkestrav1alpha1.ApplicationGroup{}
		if s, ok := appGroup.Annotations[lastSuccessfulApplicationGroupKey]; ok {
			_ = json.Unmarshal([]byte(s), last)
			r.lastSuccessfulApplicationGroup = last
		}
	}

	// handle deletes if deletion timestamp is non-zero.
	// controller-runtime cannot guarantee the order of events
	// , so it is upto us to determine the type of event
	if !appGroup.DeletionTimestamp.IsZero() {
		// If finalizer is found, remove it and requeue
		if appGroup.Finalizers != nil {
			logr.Info("cleaning up the applicationgroup resource")

			// Reverse the entire workflow to remove all the Helm Releases

			// unset the last successful spec annotation
			r.lastSuccessfulApplicationGroup = nil
			if _, ok := appGroup.Annotations[lastSuccessfulApplicationGroupKey]; ok {
				appGroup.Annotations[lastSuccessfulApplicationGroupKey] = ""
			}
			requeue = false
			_ = r.cleanupWorkflow(ctx, logr, appGroup)
			appGroup.Finalizers = nil
			_ = r.Update(ctx, &appGroup)
			return r.handleResponseAndEvent(ctx, logr, appGroup, requeue, nil)
		}
		// Do nothing
		return ctrl.Result{Requeue: false}, nil
	}

	// Initialize all the application specs and status fields embedded in the application group
	initApplications(&appGroup)

	// Add finalizer if it doesnt already exist
	if appGroup.Finalizers == nil {
		appGroup.Finalizers = []string{finalizer}
		_ = r.Update(ctx, &appGroup)
		return ctrl.Result{Requeue: true}, nil
	}

	// If the (needs) Rollback phase is present in the reconciled version,
	// we must rollback the application group to the last successful spec.
	// This should only happen on updates and not during installs.
	if appGroup.Status.Phase == orkestrav1alpha1.Rollback {
		logr.Info("Rolling back to last successful application group spec")
		appGroup.Spec = r.lastSuccessfulApplicationGroup.DeepCopy().Spec
		err = r.Update(ctx, &appGroup)
		if err != nil {
			logr.Error(err, "failed to update ApplicationGroup instance while rolling back")
			return r.handleResponseAndEvent(ctx, logr, appGroup, requeue, err)
		}
		appGroup.Status.Phase = ""
		requeue = true
		err = nil
		return r.handleResponseAndEvent(ctx, logr, appGroup, requeue, err)
	}

	// Create/Update scenario
	// Compares the current generation to the generation that was last
	// seen and updated by the reconciler
	if appGroup.Generation != appGroup.Status.ObservedGeneration {
		if appGroup.Status.ObservedGeneration != 0 {
			appGroup.Status.Update = true
		}
		requeue, err = r.reconcile(ctx, logr, r.WorkflowNS, &appGroup)
		if err != nil {
			logr.Error(err, "failed to reconcile ApplicationGroup instance")
			return r.handleResponseAndEvent(ctx, logr, appGroup, requeue, err)
		}

		appGroup.Status.ObservedGeneration = appGroup.Generation
		switch appGroup.Status.Phase {
		case orkestrav1alpha1.Init, orkestrav1alpha1.Running:
			logr.V(1).Info("workflow in init/running state. requeue and reconcile after a short period")
			requeue = true
			err = nil
		case orkestrav1alpha1.Succeeded:
			logr.V(1).Info("workflow ran to completion and succeeded")
			requeue = false
			err = nil
		case orkestrav1alpha1.Error:
			requeue = false
			err = fmt.Errorf("workflow in failure/error condition")
			logr.Error(err, "workflow in failure/error condition")
		default:
			requeue = false
			err = nil
		}

		return r.handleResponseAndEvent(ctx, logr, appGroup, requeue, err)
	}

	// Calculate the cumulative status of the generated Workflow
	// and the generated HelmRelease objects

	// Lookup Workflow by ownership and heritage labels
	// We are expecting to find at most one workflow in
	// the returned list that is associated with this
	// ApplicationGroup object.
	wfs := v1alpha12.WorkflowList{}
	listOption := client.MatchingLabels{
		workflow.OwnershipLabel: appGroup.Name,
		workflow.HeritageLabel:  workflow.Project,
	}
	err = r.List(ctx, &wfs, listOption)
	if err != nil {
		logr.Error(err, "failed to find generated workflow instance")
		requeue = false
		return r.handleResponseAndEvent(ctx, logr, appGroup, requeue, err)
	}

	if wfs.Items.Len() == 0 {
		err = fmt.Errorf("listed workflows len is 0")
		logr.Error(err, "no associated workflow found")
		requeue = false
		return r.handleResponseAndEvent(ctx, logr, appGroup, requeue, err)
	}

	// determine the associated/generated workflow status
	var phase orkestrav1alpha1.ReconciliationPhase
	wfStatus := wfs.Items[0].Status.Phase
	switch wfStatus {
	case v1alpha12.NodeError, v1alpha12.NodeFailed:
		phase = orkestrav1alpha1.Error
	case v1alpha12.NodePending, v1alpha12.NodeRunning:
		phase = orkestrav1alpha1.Running
	case v1alpha12.NodeSucceeded:
		phase = orkestrav1alpha1.Succeeded
		// case v1alpha12.NodeSkipped:
		// 	phase = orkestrav1alpha1.Succeeded
	}

	appGroup.Status.Phase = phase

	helmReleaseStatusMap := make(map[string]helmopv1.HelmReleasePhase)

	// XXX (nitishm) Not sure why this happens ???
	// Lookup all associated HelmReleases for status as well since the Workflow will not always reflect the status of the HelmRelease
	// Lookup Workflow by ownership and heritage labels
	helmReleases := helmopv1.HelmReleaseList{}
	err = r.List(ctx, &helmReleases, listOption)
	if err != nil {
		logr.Error(err, "failed to find generated HelmRelease instance")
		requeue = false
		return r.handleResponseAndEvent(ctx, logr, appGroup, requeue, err)
	}

	for _, hr := range helmReleases.Items {
		name := hr.Name
		if v, ok := hr.GetAnnotations()["orkestra/parent-chart"]; ok {
			// Use the parent charts name
			name = v
		}
		// XXX (nitishm) Needs more thought and testing
		if _, ok := helmReleaseStatusMap[name]; ok {
			if hr.Status.Phase != helmopv1.HelmReleasePhaseSucceeded {
				helmReleaseStatusMap[name] = hr.Status.Phase
			}
		} else {
			helmReleaseStatusMap[name] = hr.Status.Phase
		}
	}

	// Update each application status using the HelmRelease status
	v := make([]orkestrav1alpha1.ApplicationStatus, 0)
	for _, app := range appGroup.Status.Applications {
		app.ChartStatus.Phase = helmReleaseStatusMap[app.Name]
		v = append(v, app)
	}
	appGroup.Status.Applications = v

	// This is the cumulative status from the workflow phase and the helmrelease object statuses
	err = componentStatus(phase, helmReleaseStatusMap)
	if err != nil {
		// Any error arising from the workflow or the helmreleases should be marked as a NodeError
		appGroup.Status.Phase = orkestrav1alpha1.Error
	}

	switch appGroup.Status.Phase {
	case orkestrav1alpha1.Running, orkestrav1alpha1.Init:
		logr.V(1).Info("workflow in init/running state. requeue and reconcile after a short period")
		requeue = true
		err = nil
	case orkestrav1alpha1.Succeeded:
		logr.V(1).Info("workflow ran to completion and succeeded")
		requeue = false
		err = nil
	case orkestrav1alpha1.Error:
		requeue = false
		err = fmt.Errorf("workflow in failure/error condition : %w", err)
		logr.Error(err, "")
	default:
		requeue = false
		err = nil
	}

	return r.handleResponseAndEvent(ctx, logr, appGroup, requeue, err)
}

func (r *ApplicationGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&orkestrav1alpha1.ApplicationGroup{}).
		WithEventFilter(pred).
		Complete(r)
}

func (r *ApplicationGroupReconciler) handleResponseAndEvent(ctx context.Context, logr logr.Logger, grp orkestrav1alpha1.ApplicationGroup, requeue bool, err error) (ctrl.Result, error) {
	var errStr string
	if err != nil {
		errStr = err.Error()
	}

	grp.Status.Error = errStr

	_ = r.Status().Update(ctx, &grp)

	if grp.Status.Error == "" && grp.Status.Phase == orkestrav1alpha1.Succeeded {
		// Annotate the resource with the last successful ApplicationGroup spec
		b, _ := json.Marshal(&grp)
		grp.SetAnnotations(map[string]string{lastSuccessfulApplicationGroupKey: string(b)})
		r.lastSuccessfulApplicationGroup = grp.DeepCopy()
		_ = r.Update(ctx, &grp)

		r.Recorder.Event(&grp, "Normal", "ReconcileSuccess", fmt.Sprintf("Successfully reconciled ApplicationGroup %s", grp.Name))
	}

	if errStr != "" {
		r.Recorder.Event(&grp, "Warning", "ReconcileError", fmt.Sprintf("Failed to reconcile ApplicationGroup %s with Error : %s", grp.Name, errStr))
	}

	if err != nil {
		if !r.DisableRemediation {
			return r.handleRemediation(ctx, logr, grp, err)
		}
	}

	if requeue {
		return reconcile.Result{RequeueAfter: getInterval(&grp)}, err
	}

	return reconcile.Result{}, err
}

func initApplications(appGroup *orkestrav1alpha1.ApplicationGroup) {
	// Initialize the Status fields if not already setup
	if len(appGroup.Status.Applications) == 0 {
		appGroup.Status.Applications = make([]orkestrav1alpha1.ApplicationStatus, 0, len(appGroup.Spec.Applications))
		for _, app := range appGroup.Spec.Applications {
			status := orkestrav1alpha1.ApplicationStatus{
				Name:        app.Name,
				ChartStatus: orkestrav1alpha1.ChartStatus{Version: app.Spec.Chart.Version},
				Subcharts:   make(map[string]orkestrav1alpha1.ChartStatus),
			}
			appGroup.Status.Applications = append(appGroup.Status.Applications, status)
		}
	}
}

func (r *ApplicationGroupReconciler) handleRemediation(ctx context.Context, logr logr.Logger, g orkestrav1alpha1.ApplicationGroup, err error) (ctrl.Result, error) {
	// Rollback to previous successful spec since the annotation was set and this is
	// an UPDATE event
	if r.lastSuccessfulApplicationGroup != nil {
		// If this is a HelmRelease failure then we must remediate by cleaning up
		// all the helm releases deployed by the workflow and helm operator
		if errors.Is(err, ErrHelmReleaseInFailureStatus) {
			// Delete the HelmRelease(s) - parent and subchart(s)
			// Lookup charts using the label selector.
			// Example: chart=kafka-dev,heritage=orkestra,owner=dev, where chart=<top-level-chart>
			logr.Info("Remediating the applicationgroup with helmrelease failure status")
			for _, app := range g.Status.Applications {
				switch app.ChartStatus.Phase {
				case helmopv1.HelmReleasePhaseFailed, helmopv1.HelmReleasePhaseDeployFailed, helmopv1.HelmReleasePhaseChartFetchFailed, helmopv1.HelmReleasePhaseTestFailed, helmopv1.HelmReleasePhaseRollbackFailed:
					listOption := client.MatchingLabels{
						workflow.OwnershipLabel: g.Name,
						workflow.HeritageLabel:  workflow.Project,
						workflow.ChartLabelKey:  app.Name,
					}
					helmReleases := helmopv1.HelmReleaseList{}
					err = r.List(ctx, &helmReleases, listOption)
					if err != nil {
						logr.Error(err, "failed to find generated HelmRelease instances")
						return reconcile.Result{Requeue: false}, err
					}

					err = r.rollbackFailedHelmReleases(ctx, helmReleases.Items)
					if err != nil {
						logr.Error(err, "failed to rollback failed HelmRelease instances")
						return reconcile.Result{Requeue: false}, err
					}
				}
			}
		}
		// mark the object as requiring rollback so that we can rollback
		// to the previous versions of all the applications in the ApplicationGroup
		// using the last successful spec
		g.Status.Phase = orkestrav1alpha1.Rollback
		_ = r.Status().Update(ctx, &g)

		return reconcile.Result{RequeueAfter: getInterval(&g)}, nil
	}
	// Reverse and cleanup the workflow and associated helmreleases
	_ = r.cleanupWorkflow(ctx, logr, g)

	return reconcile.Result{Requeue: false}, err
}

func componentStatus(phase orkestrav1alpha1.ReconciliationPhase, apps map[string]helmopv1.HelmReleasePhase) error {
	for _, v := range apps {
		switch v {
		case helmopv1.HelmReleasePhaseFailed, helmopv1.HelmReleasePhaseDeployFailed, helmopv1.HelmReleasePhaseChartFetchFailed, helmopv1.HelmReleasePhaseTestFailed, helmopv1.HelmReleasePhaseRollbackFailed:
			return ErrHelmReleaseInFailureStatus
		}
	}

	if phase == orkestrav1alpha1.Error {
		return ErrWorkflowInFailureStatus
	}

	return nil
}

func (r *ApplicationGroupReconciler) rollbackFailedHelmReleases(ctx context.Context, hrs []helmopv1.HelmRelease) error {
	for _, hr := range hrs {
		err := pkg.HelmRollback(hr.Spec.ReleaseName, hr.Spec.TargetNamespace)
		if err != nil {
			return err
		}
		err = r.Client.Delete(ctx, &hr)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *ApplicationGroupReconciler) cleanupWorkflow(ctx context.Context, logr logr.Logger, g orkestrav1alpha1.ApplicationGroup) error {
	nodes := make(map[string]v1alpha12.NodeStatus)
	wfs := v1alpha12.WorkflowList{}
	listOption := client.MatchingLabels{
		workflow.OwnershipLabel: g.Name,
		workflow.HeritageLabel:  workflow.Project,
	}
	_ = r.List(ctx, &wfs, listOption)

	if wfs.Items.Len() != 0 {
		logr.Info("Reversing the workflow")

		wf := wfs.Items[0]
		for _, node := range wf.Status.Nodes {
			nodes[node.ID] = node
		}
		graph, err := workflow.Build(g.Name, nodes)
		if err != nil {
			logr.Error(err, "failed to build the wf status DAG")
		}

		rev := graph.Reverse()

		for _, bucket := range rev {
			for _, hr := range bucket {
				err = pkg.HelmUninstall(hr.Spec.ReleaseName, hr.Spec.TargetNamespace)
				if err != nil {
					logr.Error(err, "failed to uninstall helm release using helm actions - continuing with cleanup")
				}
			}
		}

		for _, bucket := range rev {
			for _, hr := range bucket {
				err = r.Client.Delete(ctx, &hr)
				if err != nil {
					logr.Error(err, "failed to delete helmrelease CRO - continuing with cleanup")
				}
			}
		}

		err = r.Client.Delete(ctx, &wf)
		if err != nil {
			logr.Error(err, "failed to delete workflow CRO - continuing with cleanup")
		}
	}
	return nil
}

func getInterval(appGroup *orkestrav1alpha1.ApplicationGroup) time.Duration {
	if appGroup.Spec.Interval != nil {
		return appGroup.Spec.Interval.Duration
	}
	return time.Duration(30 * time.Second)
}
