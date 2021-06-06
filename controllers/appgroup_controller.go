// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/Orkestra/pkg/utils"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/Azure/Orkestra/pkg/workflow"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/go-logr/logr"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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
}

// +kubebuilder:rbac:groups=orkestra.azure.microsoft.com,resources=applicationgroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=orkestra.azure.microsoft.com,resources=applicationgroups/status,verbs=get;update;patch

func (r *ApplicationGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	appGroup := &v1alpha1.ApplicationGroup{}

	logr := r.Log.WithValues(v1alpha1.AppGroupNameKey, req.NamespacedName.Name)

	if err := r.Get(ctx, req.NamespacedName, appGroup); err != nil {
		if kerrors.IsNotFound(err) {
			logr.V(3).Info("skip reconciliation since AppGroup instance not found on the cluster")
			return ctrl.Result{}, nil
		}
		logr.Error(err, "unable to fetch ApplicationGroup instance")
		return ctrl.Result{}, err
	}

	// TODO: We should not be passing the patch around; instead, we should be deferring the patch operation
	// at the end of the reconcile loop. This needs to be validated though
	patch := client.MergeFrom(appGroup.DeepCopy())

	if !appGroup.DeletionTimestamp.IsZero() {
		if err := r.reconcileDelete(ctx, appGroup, patch); err != nil {
			return r.Failed(ctx, appGroup, patch, err)
		}
		return ctrl.Result{Requeue: true}, nil
	}
	// Add finalizer if it doesn't already exist
	if appGroup.Finalizers == nil {
		controllerutil.AddFinalizer(appGroup, v1alpha1.AppGroupFinalizer)
		if err := r.Patch(ctx, appGroup, patch); err != nil {
			return r.Failed(ctx, appGroup, patch, err)
		}
	}

	// If we have not yet seen this generation, we should reconcile and create the workflow
	// Only do this if we have successfully completed a rollback
	if appGroup.Generation != appGroup.Status.ObservedGeneration {
		// Change the app group spec into a progressing state
		if err := r.reconcileCreateOrUpdate(ctx, appGroup, patch); err != nil {
			return r.Failed(ctx, appGroup, patch, err)
		}
	}

	// While ready is progressing, we get the state of the workflow
	if appGroup.Generation != appGroup.Status.LastSucceededGeneration {
		if err := r.UpdateStatus(ctx, appGroup, patch); err != nil {
			return r.Failed(ctx, appGroup, patch, fmt.Errorf("failed to update the status of the progressing application group with err: %v", err))
		}
		return r.UpdateStatusWithWorkflow(ctx, appGroup, patch)
	}

	// If we are not progressing, update the status and requeue
	if err := r.UpdateStatus(ctx, appGroup, patch); err != nil {
		return r.Failed(ctx, appGroup, patch, fmt.Errorf("failed to update the status of the application group with err: %v", err))
	}
	return ctrl.Result{RequeueAfter: v1alpha1.GetInterval(appGroup)}, nil
}

func (r *ApplicationGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ApplicationGroup{}).
		WithEventFilter(pred).
		Complete(r)
}

func (r *ApplicationGroupReconciler) rollbackFailedHelmReleases(ctx context.Context, hrs []fluxhelmv2beta1.HelmRelease) error {
	for _, hr := range hrs {
		err := utils.HelmRollback(hr.Spec.ReleaseName, hr.Spec.TargetNamespace)
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
