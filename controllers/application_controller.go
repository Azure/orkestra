// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/configurer"
	"github.com/Azure/Orkestra/pkg/registry"
)

const (
	appNameKey = "appgroup"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	// Cfg is the controller configuration that gives access to the helm registry configuration (and more as we add options to configure the controller)
	Cfg *configurer.Controller

	// RegistryClient interacts with the helm registries to pull and push charts
	RegistryClient *registry.Client

	// StagingRepoName is the nickname for the repository used for staging artifacts before being deployed using the HelmRelease object
	StagingRepoName string

	// Recorder generates kubernetes events
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=orkestra.azure.microsoft.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=orkestra.azure.microsoft.com,resources=applications/status,verbs=get;update;patch

func (r *ApplicationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var requeue bool
	var err error
	var application orkestrav1alpha1.Application

	ctx := context.Background()
	logr := r.Log.WithValues(appNameKey, req.NamespacedName.Name)

	if err := r.Get(ctx, req.NamespacedName, &application); err != nil {
		if errors.IsNotFound(err) {
			logr.V(3).Info("skip reconciliation since Appllication instance not found on the cluster")
			return ctrl.Result{}, nil
		}
		logr.Error(err, "unable to fetch Application instance")
		return ctrl.Result{}, err
	}

	logr = logr.WithValues("status-ready", application.Status.Application.Ready, "status-error", application.Status.Application.Error)

	if application.Status.Application.Ready {
		logr.V(3).Info("skip reconciling since Application has already been successfully reconciled")
		return ctrl.Result{Requeue: false}, nil
	}

	// info log if status error is not nil on reconciling
	if application.Status.Application.Error != "" {
		logr.V(3).Info("reconciling Application instance previously in error state")
	}

	requeue, err = r.reconcile(ctx, logr, &application)
	defer r.updateStatusAndEvent(ctx, application, requeue, err)
	if err != nil {
		logr.Error(err, "failed to reconcile application instance")
		return ctrl.Result{Requeue: requeue}, err
	}

	return ctrl.Result{Requeue: requeue}, nil
}

func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&orkestrav1alpha1.Application{}).
		Complete(r)
}

func (r *ApplicationReconciler) updateStatusAndEvent(ctx context.Context, app orkestrav1alpha1.Application, requeue bool, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	app.Status.Application.Ready = !requeue
	app.Status.Application.Error = errStr

	_ = r.Status().Update(ctx, &app)

	if errStr != "" {
		r.Recorder.Event(&app, "Warning", "ReconcileError", fmt.Sprintf("Failed to reconcile Application %s with Error %s", app.Name, errStr))
	} else {
		r.Recorder.Event(&app, "Normal", "ReconcileSuccess", fmt.Sprintf("Successfully reconciled Application %s", app.Name))
	}
}
