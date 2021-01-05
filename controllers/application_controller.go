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
	_ = context.Background()
	_ = r.Log.WithValues("application", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
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

	app.Status = orkestrav1alpha1.ApplicationStatus{}

	_ = r.Status().Update(ctx, &app)

	if errStr != "" {
		r.Recorder.Event(&app, "Warning", "ReconcileError", fmt.Sprintf("Failed to reconcile Application %s with Error %s", app.Name, errStr))
	} else {
		r.Recorder.Event(&app, "Normal", "ReconcileSuccess", fmt.Sprintf("Successfully reconciled Application %s", app.Name))
	}
}
