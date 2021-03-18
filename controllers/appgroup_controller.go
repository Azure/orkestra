// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/Orkestra/pkg"
	"github.com/Azure/Orkestra/pkg/configurer"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/Azure/Orkestra/pkg/workflow"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	helmopv1 "github.com/fluxcd/helm-operator/pkg/apis/helm.fluxcd.io/v1"
	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/chart"
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
	requeueAfter                      = 5 * time.Second
	requeueAfterLong                  = requeueAfter * 6
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

	// Cfg is the controller configuration that gives access to the helm registry configuration (and more as we add options to configure the controller)
	Cfg    *configurer.Controller
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

	if appGroup.GetAnnotations() != nil {
		last := &orkestrav1alpha1.ApplicationGroup{}
		if s, ok := appGroup.Annotations[lastSuccessfulApplicationGroupKey]; ok {
			_ = json.Unmarshal([]byte(s), last)
			r.lastSuccessfulApplicationGroup = last
		}
	}

	// handle deletes if deletion timestamp is non-zero
	if !appGroup.DeletionTimestamp.IsZero() {
		// If finalizer is found, remove it and requeue
		if appGroup.Finalizers != nil {
			defer func() {
				appGroup.Finalizers = nil
				_ = r.Update(ctx, &appGroup)
			}()

			logr.Info("cleaning up the applicationgroup resource")

			// Reverse the entire workflow to remove all the Helm Releases
			r.lastSuccessfulApplicationGroup = nil
			if _, ok := appGroup.Annotations[lastSuccessfulApplicationGroupKey]; ok {
				appGroup.Annotations[lastSuccessfulApplicationGroupKey] = ""
			}
			requeue = false
			err = r.cleanupWorkflow(ctx, logr, appGroup)
			if err != nil {
				logr.Error(err, "failed to clean up workflow")
				return r.handleResponseAndEvent(ctx, logr, appGroup, requeue, err)
			}
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

	// handle first time install and subsequent updates
	checksums, err := pkg.Checksum(&appGroup)
	if err != nil {
		if errors.Is(err, pkg.ErrChecksumAppGroupSpecMismatch) {
			if appGroup.Status.Checksums != nil {
				appGroup.Status.Update = true
			}
			requeue, err = r.reconcile(ctx, logr, r.WorkflowNS, &appGroup)
			if err != nil {
				logr.Error(err, "failed to reconcile ApplicationGroup instance")
				return r.handleResponseAndEvent(ctx, logr, appGroup, requeue, err)
			}

			appGroup.Status.Checksums = checksums

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

		logr.Error(err, "failed to calculate checksum annotations for application group specs")
		return r.handleResponseAndEvent(ctx, logr, appGroup, requeue, err)
	}

	appGroup.Status.Checksums = checksums

	// Lookup Workflow by ownership and heritage labels
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
		// appGroup.Status.Phase = wfs.Items[0].Status.Phase
		err = fmt.Errorf("listed workflows len is 0")
		logr.Error(err, "no associated workflow found")
		requeue = false
		return r.handleResponseAndEvent(ctx, logr, appGroup, requeue, err)
	}

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

	grp.Status.Error = errStr

	_ = r.Status().Update(ctx, &grp)

	if grp.Status.Phase == orkestrav1alpha1.Succeeded {
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
		return r.handleRemediation(ctx, logr, grp, err)
	}

	if requeue {
		interval := requeueAfter
		if grp.Status.Phase != orkestrav1alpha1.Running {
			interval = requeueAfterLong
		}
		return reconcile.Result{Requeue: true, RequeueAfter: interval}, err
	}

	return reconcile.Result{Requeue: requeue}, err
}

func isDependenciesEmbedded(ch *chart.Chart) bool {
	// TODO (nitishm) This does not support a mix of remote and embedded dependency subcharts
	isURI := false
	for _, d := range ch.Metadata.Dependencies {
		if _, err := url.ParseRequestURI(d.Repository); err == nil {
			// If this is an "assembled" chart (https://helm.sh/docs/chart_best_practices/dependencies/#versions) we must stage the embedded subchart
			if strings.Contains(d.Repository, "file://") {
				isURI = false
				break
			}
			isURI = true
		}
	}

	if !isURI {
		if len(ch.Dependencies()) > 0 {
			return true
		}
	}
	return false
}

func initApplications(appGroup *orkestrav1alpha1.ApplicationGroup) {
	// Initialize the Status fields if not already setup
	if len(appGroup.Status.Applications) == 0 {
		appGroup.Status.Applications = make([]orkestrav1alpha1.ApplicationStatus, 0, len(appGroup.Spec.Applications))
		for _, app := range appGroup.Spec.Applications {
			status := orkestrav1alpha1.ApplicationStatus{
				Name:        app.Name,
				ChartStatus: orkestrav1alpha1.ChartStatus{Version: app.Spec.Version},
				Subcharts:   make(map[string]orkestrav1alpha1.ChartStatus),
			}
			appGroup.Status.Applications = append(appGroup.Status.Applications, status)
		}
	}

	// Initialize fields in the Application spec for every app in the appgroup
	v := orkestrav1alpha1.ApplicationGroup{}
	for _, app := range appGroup.Spec.Applications {
		if app.Spec.Overlays.Data == nil {
			app.Spec.Overlays.Data = make(map[string]interface{})
		}
		app.Spec.Values = app.Spec.Overlays
		v.Spec.Applications = append(v.Spec.Applications, app)
	}
	appGroup.Spec.Applications = v.DeepCopy().Spec.Applications
}

func (r *ApplicationGroupReconciler) handleRemediation(ctx context.Context, logr logr.Logger, g orkestrav1alpha1.ApplicationGroup, err error) (ctrl.Result, error) {
	// Rollback to previous successful spec
	if r.lastSuccessfulApplicationGroup != nil {
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

					g.Status.Phase = orkestrav1alpha1.Rollback
					_ = r.Status().Update(ctx, &g)

					return reconcile.Result{Requeue: true, RequeueAfter: requeueAfter}, nil
				}
			}
		}
		return reconcile.Result{Requeue: false}, err
	}
	// Reverse and cleanup the workflow and associated helmreleases
	err2 := r.cleanupWorkflow(ctx, logr, g)
	if err2 != nil {
		err = fmt.Errorf("failed to clean up workflow : %w", err2)
	}

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
	err := r.List(ctx, &wfs, listOption)
	if err != nil {
		return err
	}

	if wfs.Items.Len() != 0 {
		logr.Info("Reversing the workflow")

		wf := wfs.Items[0]
		for _, node := range wf.Status.Nodes {
			nodes[node.ID] = node
		}
		graph, err := workflow.Build(g.Name, nodes)
		if err != nil {
			logr.Error(err, "failed to build the wf status DAG")
			return err
		}

		rev := graph.Reverse()

		for _, bucket := range rev {
			for _, hr := range bucket {
				err = pkg.HelmUninstall(hr.Spec.ReleaseName, hr.Spec.TargetNamespace)
				if err != nil {
					logr.Error(err, "failed to uninstall helm release using helm actions")
				}
				err = r.Client.Delete(ctx, &hr)
				if err != nil {
					return err
				}
				// time.Sleep(time.Second * 2)
				// TODO (nitishm) Use the helm package to delete the release and wait for it to be cleaned up
			}
		}

		err = r.Client.Delete(ctx, &wf)
		if err != nil {
			return err
		}
	}
	return nil
}
