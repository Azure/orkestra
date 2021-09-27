// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package controllers

import (
	"context"
	"errors"
	"github.com/Azure/Orkestra/pkg/meta"

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

	// RegistryClient interacts with the helm registries to pull and push charts
	RegistryClient *registry.Client

	// StagingRepoName is the nickname for the repository used for staging artifacts before being deployed using the HelmRelease object
	StagingRepoName string

	WorkflowClientBuilder *workflow.Builder

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

func (r *ApplicationGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
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

	statusHelper := &helpers.StatusHelper{
		Client:    r.Client,
		Logger:    logr,
		PatchFrom: patch,
		Recorder:  r.Recorder,
	}
	reconcileHelper := helpers.ReconcileHelper{
		Client:                r.Client,
		Logger:                logr,
		Instance:              appGroup,
		WorkflowClientBuilder: r.WorkflowClientBuilder,
		RegistryClient:        r.RegistryClient,
		RegistryOptions: helpers.RegistryClientOptions{
			StagingRepoName:         r.StagingRepoName,
			TargetDir:               r.TargetDir,
			CleanupDownloadedCharts: r.CleanupDownloadedCharts,
		},
		StatusHelper: statusHelper,
	}

	// Patch the status before returning from the reconcile loop
	defer func() {
		// Update the err value which is scoped outside the defer
		patchErr := statusHelper.PatchStatus(ctx, appGroup)
		if err == nil {
			err = patchErr
		}
	}()

	if !appGroup.DeletionTimestamp.IsZero() {
		statusHelper.MarkTerminating(appGroup)
		if err := reconcileHelper.Reverse(ctx); errors.Is(err, meta.ErrForwardWorkflowNotFound) {
			controllerutil.RemoveFinalizer(appGroup, v1alpha1.AppGroupFinalizer)
			if err := r.Patch(ctx, appGroup, patch); err != nil {
				logr.Error(err, "failed to patch the release to remove the appgroup finalizer")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			logr.Error(err, "failed to generate the reverse workflow")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	// Add finalizer if it doesn't already exist
	if !controllerutil.ContainsFinalizer(appGroup, v1alpha1.AppGroupFinalizer) {
		controllerutil.AddFinalizer(appGroup, v1alpha1.AppGroupFinalizer)
		if err := r.Patch(ctx, appGroup, patch); err != nil {
			logr.Error(err, "failed to patch the release with the appgroup finalizer")
			return ctrl.Result{}, err
		}
	}

	// If we have not yet seen this generation, we should reconcile and create the workflow
	// Only do this if we have successfully completed a rollback
	if appGroup.Generation != appGroup.Status.ObservedGeneration {
		// Change the app group spec into a progressing state
		statusHelper.MarkProgressing(appGroup)
		if err := reconcileHelper.CreateOrUpdate(ctx); err != nil {
			logr.Error(err, "failed to reconcile creating or updating the appgroup")
			return ctrl.Result{}, err
		}
		appGroup.Status.ObservedGeneration = appGroup.Generation
	}
	return ctrl.Result{}, nil
}

func (r *ApplicationGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ApplicationGroup{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
