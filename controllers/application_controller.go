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
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

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
	logr := r.Log.WithValues("application", req.NamespacedName)

	if err := r.Get(ctx, req.NamespacedName, &application); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logr.Error(err, "unable to fetch Application")
		return ctrl.Result{}, err
	}

	if application.Status.ChartStatus.Ready {
		return ctrl.Result{}, nil
	}

	requeue, err = r.reconcile(logr, &application)
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

	app.Status = orkestrav1alpha1.ApplicationStatus{
		Name: app.Name,
		ChartStatus: orkestrav1alpha1.ChartStatus{
			Ready: !requeue,
			Error: errStr,
		},
	}

	_ = r.Status().Update(ctx, &app)

	if errStr != "" {
		r.Recorder.Event(&app, "Warning", "ReconcileError", fmt.Sprintf("Failed to reconcile Application %s with Error %s", app.Name, errStr))
	} else {
		r.Recorder.Event(&app, "Normal", "ReconcileSuccess", fmt.Sprintf("Successfully reconciled Application %s", app.Name))
	}
}
