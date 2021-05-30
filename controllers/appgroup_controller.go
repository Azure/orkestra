// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Azure/Orkestra/pkg/utils"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/Azure/Orkestra/pkg/meta"
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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/Orkestra/api/v1alpha1"
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

	// lastSuccessfulApplicationGroup holds the applicationgroup spec body from the last
	// successful reconciliation of the ApplicationGroup. This is set after every successful
	// reconciliation.
	lastSuccessfulApplicationGroup *v1alpha1.ApplicationGroup
}

// +kubebuilder:rbac:groups=orkestra.azure.microsoft.com,resources=applicationgroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=orkestra.azure.microsoft.com,resources=applicationgroups/status,verbs=get;update;patch

func (r *ApplicationGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var requeue bool
	var err error
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

	// Check if this is an update event to the ApplicationGroup
	// in which case unmarshal the last successful spec into a
	// variable
	if appGroup.GetAnnotations() != nil {
		last := &v1alpha1.ApplicationGroup{}
		if s, ok := appGroup.Annotations[v1alpha1.LastSuccessfulAnnotation]; ok {
			_ = json.Unmarshal([]byte(s), last)
			r.lastSuccessfulApplicationGroup = last
		}
	}
	if !appGroup.DeletionTimestamp.IsZero() {
		if err := r.reconcileDelete(ctx, appGroup, patch); err != nil {
			return r.Failed(ctx, appGroup, patch, err)
		}
		return ctrl.Result{Requeue: true}, nil
	}
	// Add finalizer if it doesnt already exist
	if appGroup.Finalizers == nil {
		controllerutil.AddFinalizer(appGroup, v1alpha1.AppGroupFinalizer)
		if err := r.Patch(ctx, appGroup, patch); err != nil {
			return r.Failed(ctx, appGroup, patch, err)
		}
	}

	// If the (needs) Rollback phase is present in the reconciled version,
	// we must rollback the application group to the last successful spec.
	// This should only happen on updates and not during installs.
	if appGroup.GetDeployCondition() == meta.RollingBackReason {
		if err := r.reconcileRollback(ctx, appGroup, patch); err != nil {
			return r.Failed(ctx, appGroup, patch, err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if appGroup.Generation != appGroup.Status.ObservedGeneration {
		// Change the app group spec into a progressing state
		requeue, err = r.reconcileCreateOrUpdate(ctx, appGroup, patch)
		if err != nil {
			return r.Failed(ctx, appGroup, patch, err)
		}
	}
	if appGroup.GetReadyCondition() == meta.FailedReason {
		err = fmt.Errorf("workflow in failure/error condition")
		logr.Error(err, "workflow in failure/error condition")
		return r.Remediate(ctx, appGroup, patch, err)
	}
	return r.UpdateStatus(ctx, appGroup, patch, requeue)
}

func (r *ApplicationGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ApplicationGroup{}).
		WithEventFilter(pred).
		Complete(r)
}

func (r *ApplicationGroupReconciler) handleRemediation(ctx context.Context, logr logr.Logger, g *v1alpha1.ApplicationGroup,
	patch client.Patch, err error) (ctrl.Result, error) {
	// Rollback to previous successful spec since the annotation was set and this is
	// an UPDATE event
	if r.lastSuccessfulApplicationGroup != nil {
		// If this is a HelmRelease failure then we must remediate by cleaning up
		// all the helm releases deployed by the workflow and helm operator
		if errors.Is(err, ErrHelmReleaseInFailureStatus) {
			// Delete the HelmRelease(s) - parent and subchart(s)
			// Lookup charts using the label selector.
			// Example: chart=kafka-dev,heritage=orkestra,owner=dev, where chart=<top-level-chart>
			logr.Info("Remediating the applicationgroup with helmrelease failure status")
			for _, app := range g.Status.Applications {
				switch meta.GetResourceCondition(&app.ChartStatus, meta.ReadyCondition).Reason {
				case fluxhelmv2beta1.InstallFailedReason, fluxhelmv2beta1.UpgradeFailedReason, fluxhelmv2beta1.UninstallFailedReason,
					fluxhelmv2beta1.ArtifactFailedReason, fluxhelmv2beta1.InitFailedReason, fluxhelmv2beta1.GetLastReleaseFailedReason:
					listOption := client.MatchingLabels{
						workflow.OwnershipLabel: g.Name,
						workflow.HeritageLabel:  workflow.Project,
						workflow.ChartLabelKey:  app.Name,
					}
					helmReleases := fluxhelmv2beta1.HelmReleaseList{}
					err = r.List(ctx, &helmReleases, listOption)
					if err != nil {
						logr.Error(err, "failed to find generated HelmRelease instances")
						return reconcile.Result{}, nil
					}

					err = r.rollbackFailedHelmReleases(ctx, helmReleases.Items)
					if err != nil {
						logr.Error(err, "failed to rollback failed HelmRelease instances")
						return reconcile.Result{}, nil
					}
				}
			}
		}
		// mark the object as requiring rollback so that we can rollback
		// to the previous versions of all the applications in the ApplicationGroup
		// using the last successful spec
		g.RollingBack()
		_ = r.Status().Patch(ctx, g, patch)
		logr.WithValues("requeueTime", v1alpha1.DefaultProgressingRequeue.String())
		logr.V(1).Info("initiating rollback")
		return reconcile.Result{RequeueAfter: v1alpha1.DefaultProgressingRequeue}, nil
	}
	// Reverse and cleanup the workflow and associated helmreleases
	g.RollingBack()
	_ = r.Status().Patch(ctx, g, patch)

	requeue := r.cleanupWorkflow(ctx, logr, g)
	if requeue {
		logr.Info("reverse workflow is in progress")
		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
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
