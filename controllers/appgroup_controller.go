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
	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/chart"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
)

const (
	appgroupNameKey                   = "appgroup"
	finalizer                         = "application-group-finalizer"
	requeueAfter                      = 5 * time.Second
	lastSuccessfulApplicationGroupKey = "orkestra/last-successful-applicationgroup"
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
		s := appGroup.Annotations[lastSuccessfulApplicationGroupKey]
		_ = json.Unmarshal([]byte(s), last)
		r.lastSuccessfulApplicationGroup = last
	}

	// handle deletes if deletion timestamp is non-zero
	if !appGroup.DeletionTimestamp.IsZero() {
		// If finalizer is found, remove it and requeue
		if appGroup.Finalizers != nil {
			logr.Info("cleaning up the applicationgroup resource")
			// TODO: Take remediation action
			// Reverse the entire workflow to remove all the Helm Releases
			appGroup.Finalizers = nil
			_ = r.Update(ctx, &appGroup)
			return ctrl.Result{Requeue: true}, nil
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

	// handle first time install and subsequent updates
	checksums, err := pkg.Checksum(&appGroup)
	if err != nil {
		// TODO (nitishm) Handle different error types here to decide remediation action
		if errors.Is(err, pkg.ErrChecksumAppGroupSpecMismatch) {
			if appGroup.Status.Checksums != nil {
				appGroup.Status.Update = true
			}
			requeue, err = r.reconcile(ctx, logr, r.WorkflowNS, &appGroup)
			if err != nil {
				logr.Error(err, "failed to reconcile ApplicationGroup instance")
				r.handleResponseAndEvent(ctx, appGroup, requeue, err)
				return ctrl.Result{Requeue: requeue}, err
			}

			appGroup.Status.Checksums = checksums

			if appGroup.Status.Phase != v1alpha12.NodeSucceeded {
				r.handleResponseAndEvent(ctx, appGroup, requeue, err)
				return ctrl.Result{Requeue: true, RequeueAfter: requeueAfter}, nil
			}

			r.handleResponseAndEvent(ctx, appGroup, requeue, err)
			return ctrl.Result{Requeue: requeue}, err
		}

		logr.Error(err, "failed to calculate checksum annotations for application group specs")
		r.handleResponseAndEvent(ctx, appGroup, false, err)
		return ctrl.Result{Requeue: false}, err
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
		logr.Error(err, "failed to find generate workflow instance")
		r.handleResponseAndEvent(ctx, appGroup, false, err)
		return ctrl.Result{Requeue: false}, err
	}

	if wfs.Items.Len() > 0 {
		appGroup.Status.Phase = wfs.Items[0].Status.Phase
	}

	logr = logr.WithValues("phase", appGroup.Status.Phase, "status-error", appGroup.Status.Error)

	switch appGroup.Status.Phase {
	case v1alpha12.NodeRunning, v1alpha12.NodePending:
		logr.V(1).Info("workflow in pending/running state. requeue and reconcile after a short period")
		r.handleResponseAndEvent(ctx, appGroup, true, nil)
		return ctrl.Result{Requeue: true, RequeueAfter: requeueAfter}, nil
	case v1alpha12.NodeSucceeded:
		logr.V(1).Info("workflow ran to completion and succeeded")
		r.handleResponseAndEvent(ctx, appGroup, false, nil)
		return ctrl.Result{Requeue: false}, nil
	case v1alpha12.NodeError, v1alpha12.NodeFailed:
		err = fmt.Errorf("workflow in failure/error condition")
		logr.Error(err, "workflow in failure/error condition")
		r.handleResponseAndEvent(ctx, appGroup, false, err)
		return ctrl.Result{Requeue: false}, err
	}

	return ctrl.Result{Requeue: false}, nil
}

func (r *ApplicationGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&orkestrav1alpha1.ApplicationGroup{}).
		WithEventFilter(pred).
		Complete(r)
}

func (r *ApplicationGroupReconciler) handleResponseAndEvent(ctx context.Context, grp orkestrav1alpha1.ApplicationGroup, requeue bool, err error) {
	errStr := ""
	if err != nil {
		// Handle the error by remediating the workflow
		r.handleRemediation(ctx, err)
		errStr = err.Error()
	}

	grp.Status.Error = errStr

	_ = r.Status().Update(ctx, &grp)

	if grp.Status.Phase == v1alpha12.NodeSucceeded {
		// Annotate the resource with the last successful ApplicationGroup spec
		b, _ := json.Marshal(&grp)
		grp.SetAnnotations(map[string]string{lastSuccessfulApplicationGroupKey: string(b)})
		_ = r.Update(ctx, &grp)

		r.Recorder.Event(&grp, "Normal", "ReconcileSuccess", fmt.Sprintf("Successfully reconciled ApplicationGroup %s", grp.Name))
	}

	if errStr != "" {
		r.Recorder.Event(&grp, "Warning", "ReconcileError", fmt.Sprintf("Failed to reconcile ApplicationGroup %s with Error %s", grp.Name, errStr))
	}
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

func (r *ApplicationGroupReconciler) handleRemediation(ctx context.Context, err error) {
	if r.lastSuccessfulApplicationGroup != nil {
		r.lastSuccessfulApplicationGroup.Status.Checksums = nil
		_ = r.Update(ctx, r.lastSuccessfulApplicationGroup)
	}
}
