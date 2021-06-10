// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package controllers

import (
	"context"
	"fmt"

	"github.com/Azure/Orkestra/pkg/helpers"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/Azure/Orkestra/pkg/workflow"
	"github.com/go-logr/logr"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ApplicationGroupReconciler reconciles a ApplicationGroup object
type ApplicationGroupReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	Engine workflow.Engine

	// RegistryClient interacts with the helm registries to pull and push charts
	RegistryClient *registry.Client

	Recorder record.EventRecorder

	// StagingRepoName is the nickname for the repository used for staging artifacts before being deployed using the HelmRelease object
	StagingRepoName string

	// TargetDir to stage the charts before pushing
	TargetDir string

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
	patch := client.MergeFrom(appGroup.DeepCopy())

	statusHelper := helpers.StatusHelper{
		Client:    r.Client,
		Logger:    logr,
		PatchFrom: patch,
		Recorder:  r.Recorder,
	}
	reconcileHelper := helpers.ReconcileHelper{
		Client:         r.Client,
		Logger:         logr,
		Instance:       appGroup,
		Engine:         r.Engine,
		RegistryClient: r.RegistryClient,
		RegistryOptions: helpers.RegistryClientOptions{
			StagingRepoName:         r.StagingRepoName,
			TargetDir:               r.TargetDir,
			CleanupDownloadedCharts: r.CleanupDownloadedCharts,
		},
	}

	if !appGroup.DeletionTimestamp.IsZero() {
		if err := statusHelper.MarkReversing(ctx, appGroup); err != nil {
			logr.Error(err, "failed to mark the app group into a reversing state")
			return statusHelper.Failed(ctx, appGroup, err)
		}
		result, err := reconcileHelper.Reverse(ctx)
		if !result.Requeue && err != nil {
			// Remove the finalizer because we have finished reversing
			controllerutil.RemoveFinalizer(appGroup, v1alpha1.AppGroupFinalizer)
			if err := r.Patch(ctx, appGroup, patch); err != nil {
				return ctrl.Result{}, nil
			}
		}
		return result, err
	}
	// Add finalizer if it doesn't already exist
	if appGroup.Finalizers == nil {
		controllerutil.AddFinalizer(appGroup, v1alpha1.AppGroupFinalizer)
		if err := r.Patch(ctx, appGroup, patch); err != nil {
			logr.Error(err, "failed to patch the release with the appgroup finalizer")
			return statusHelper.Failed(ctx, appGroup, err)
		}
	}

	// If we have not yet seen this generation, we should reconcile and create the workflow
	// Only do this if we have successfully completed a rollback
	if appGroup.Generation != appGroup.Status.ObservedGeneration {
		// Change the app group spec into a progressing state
		if err := statusHelper.Progressing(ctx, appGroup); err != nil {
			logr.Error(err, "failed to patch the status into a progressing state")
			return statusHelper.Failed(ctx, appGroup, err)
		}
		if err := reconcileHelper.CreateOrUpdate(ctx); err != nil {
			logr.Error(err, "failed to reconcile creating or updating the appgroup")
			return statusHelper.Failed(ctx, appGroup, err)
		}
	}

	// Update the status based on the current state of the helm charts
	if err := statusHelper.UpdateStatus(ctx, appGroup); err != nil {
		logr.Error(err, "failed to update the status of the app group")
		return statusHelper.Failed(ctx, appGroup, fmt.Errorf("failed to update the status of the progressing application group with err: %v", err))
	}

	requeueDuration := v1alpha1.GetInterval(appGroup)
	var shouldRemediate bool
	var err error

	// While ready is progressing, we get the state of the workflow
	if appGroup.Generation != appGroup.Status.LastSucceededGeneration {
		shouldRemediate, requeueDuration, err = statusHelper.UpdateStatusWithWorkflow(ctx, appGroup)
		if err != nil {
			logr.Error(err, "failed to update the status based on the workflow status")
			return statusHelper.Failed(ctx, appGroup, err)
		}
		if !r.DisableRemediation && shouldRemediate {
			if lastSuccessfulSpec := appGroup.GetLastSuccessful(); lastSuccessfulSpec != nil {
				if err := statusHelper.RollingBack(ctx, appGroup); err != nil {
					logr.Error(err, "failed to mark the app group status as rolling back")
					return statusHelper.Failed(ctx, appGroup, err)
				}
				return reconcileHelper.Rollback(ctx, patch, fmt.Errorf(""))
			}
			if err := statusHelper.MarkReversing(ctx, appGroup); err != nil {
				logr.Error(err, "failed to mark the app group status as reversing")
				return statusHelper.Failed(ctx, appGroup, err)
			}
			return reconcileHelper.Reverse(ctx)
		}
	}
	return ctrl.Result{RequeueAfter: requeueDuration}, nil
}

func (r *ApplicationGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ApplicationGroup{}).
		WithEventFilter(pred).
		Complete(r)
}
