// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package controllers

import (
	"context"
	"fmt"

	"github.com/Azure/Orkestra/pkg/configurer"
	"github.com/Azure/Orkestra/pkg/workflow"
	"github.com/go-logr/logr"
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

	// WorkflowNS is the namespace to which (generated) Argo Workflow object is deployed
	WorkflowNS string

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

	logr = logr.WithValues("status-ready", appGroup.Status.Ready, "status-error", appGroup.Status.Error)

	if appGroup.Status.Ready {
		logr.V(3).Info("skip reconciling since AppGroup has already been successfully reconciled")
		return ctrl.Result{Requeue: false}, nil
	}

	// info log if status error is not nil on reconciling
	if appGroup.Status.Error != "" {
		logr.V(3).Info("reconciling AppGroup instance previously in error state")
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&orkestrav1alpha1.ApplicationGroup{}).
		Complete(r)
}

func (r *ApplicationGroupReconciler) updateStatusAndEvent(ctx context.Context, grp orkestrav1alpha1.ApplicationGroup, requeue bool, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	grp.Status = orkestrav1alpha1.ApplicationGroupStatus{}

	_ = r.Status().Update(ctx, &grp)

	if errStr != "" {
		r.Recorder.Event(&grp, "Warning", "ReconcileError", fmt.Sprintf("Failed to reconcile ApplicationGroup %s with Error %s", grp.Name, errStr))
	} else {
		r.Recorder.Event(&grp, "Normal", "ReconcileSuccess", fmt.Sprintf("Successfully reconciled ApplicationGroup %s", grp.Name))
	}
}
