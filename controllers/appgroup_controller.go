// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
)

// ApplicationGroupReconciler reconciles a ApplicationGroup object
type ApplicationGroupReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	// Recorder generates kubernetes events
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=orkestra.azure.microsoft.com,resources=applicationgroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=orkestra.azure.microsoft.com,resources=applicationgroups/status,verbs=get;update;patch

func (r *ApplicationGroupReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("applicationgroup", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
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
