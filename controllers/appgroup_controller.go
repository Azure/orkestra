// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package controllers

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/Orkestra/pkg"
	"github.com/Azure/Orkestra/pkg/configurer"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/Azure/Orkestra/pkg/workflow"
	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
)

const (
	appgroupNameKey = "appgroup"
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
		if errors.IsNotFound(err) {
			logr.V(3).Info("skip reconciliation since AppGroup instance not found on the cluster")
			return ctrl.Result{}, nil
		}
		logr.Error(err, "unable to fetch ApplicationGroup instance")
		return ctrl.Result{}, err
	}

	_, checksums, err := pkg.Checksum(&appGroup)
	if err != nil {
		// TODO (nitishm) Handle different error types here to decide remediation action
		if checksums != nil {
			appGroup.Status.Checksums = checksums
		}
		_ = r.Status().Update(ctx, &appGroup)
		logr.V(3).Info("failed to calculate checksum annotations for application group specs")
		return ctrl.Result{Requeue: false}, err
	}

	logr = logr.WithValues("status-ready", appGroup.Status.Ready, "status-error", appGroup.Status.Error)

	appGroup.Status.Checksums = checksums

	if appGroup.Status.Ready {
		logr.V(3).Info("skip reconciling since AppGroup has already been successfully reconciled")
		return ctrl.Result{Requeue: false}, nil
	}

	// info log if status error is not nil on reconciling
	if appGroup.Status.Error != "" {
		logr.V(3).Info("reconciling AppGroup instance previously in error state")
	}

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

	requeue, err = r.reconcile(ctx, logr, r.WorkflowNS, &appGroup)
	defer r.updateStatusAndEvent(ctx, appGroup, requeue, err)
	if err != nil {
		logr.Error(err, "failed to reconcile ApplicationGroup instance")
		return ctrl.Result{Requeue: requeue}, err
	}

	return ctrl.Result{Requeue: requeue}, nil
}

func (r *ApplicationGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&orkestrav1alpha1.ApplicationGroup{}).
		WithEventFilter(pred).
		Complete(r)
}

func (r *ApplicationGroupReconciler) updateStatusAndEvent(ctx context.Context, grp orkestrav1alpha1.ApplicationGroup, requeue bool, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	grp.Status.Error = errStr

	_ = r.Status().Update(ctx, &grp)

	if errStr != "" {
		r.Recorder.Event(&grp, "Warning", "ReconcileError", fmt.Sprintf("Failed to reconcile ApplicationGroup %s with Error %s", grp.Name, errStr))
	} else {
		r.Recorder.Event(&grp, "Normal", "ReconcileSuccess", fmt.Sprintf("Successfully reconciled ApplicationGroup %s", grp.Name))
	}
}

func isDependenciesEmbedded(ch *chart.Chart) bool {
	isURI := false
	for _, d := range ch.Metadata.Dependencies {
		if _, err := url.ParseRequestURI(d.Repository); err == nil {
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
