// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package controllers

import (
	"context"
	"fmt"
	"github.com/Azure/Orkestra/pkg/meta"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/Azure/Orkestra/pkg/helpers"

	"github.com/Azure/Orkestra/api/v1alpha1"
	workflowpkg "github.com/Azure/Orkestra/pkg/workflow"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WorkflowStatusReconciler reconciles workflows and their status
type WorkflowStatusReconciler struct {
	client.Client
	Log                   logr.Logger
	Scheme                *runtime.Scheme
	WorkflowClientBuilder *workflowpkg.Builder

	// Recorder generates kubernetes events
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=argoproj.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=argoproj.io,resources=workflows/status,verbs=get;update;patch

func (r *WorkflowStatusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	workflow := &v1alpha13.Workflow{}
	parent := &v1alpha1.ApplicationGroup{}
	logr := r.Log.WithValues("workflow_name", req.NamespacedName.Name)

	if err := r.Get(ctx, req.NamespacedName, workflow); err != nil {
		if errors.IsNotFound(err) {
			logr.V(3).Info("workflow not found in the cluster")
			return ctrl.Result{}, nil
		}
		logr.Error(err, "failed to fetch workflow instance")
		return ctrl.Result{}, err
	}
	labels := workflow.GetLabels()
	if appGroupName, ok := labels[v1alpha1.OwnershipLabel]; ok {
		if err := r.Get(ctx, types.NamespacedName{Name: appGroupName}, parent); err != nil {
			logr.Error(err, "failed to get the workflow application group parent")
			return ctrl.Result{}, err
		}
	}
	workflowType, ok := labels[v1alpha1.WorkflowTypeLabel]
	if !ok {
		err := fmt.Errorf("workflow type label not found for the child workflow")
		logr.Error(err, "failed to get the workflow type label")
		return ctrl.Result{}, err
	}

	patch := client.MergeFrom(parent.DeepCopy())
	statusHelper := &helpers.StatusHelper{
		Client:    r.Client,
		Logger:    logr,
		PatchFrom: patch,
		Recorder:  r.Recorder,
	}
	reconcileHelper := helpers.ReconcileHelper{
		Client:                r.Client,
		Logger:                logr,
		Instance:              parent,
		WorkflowClientBuilder: r.WorkflowClientBuilder,
		StatusHelper:          statusHelper,
	}

	// Patch the status before returning from the reconcile loop
	defer func() {
		// Update the err value which is scoped outside the defer
		patchErr := statusHelper.PatchStatus(ctx, parent)
		if err == nil {
			err = patchErr
		}
	}()

	if parent.Generation != parent.Status.ObservedGeneration {
		return ctrl.Result{}, nil
	}

	// Remove the finalizer from the parent application group if we have just completed reversing
	if !parent.DeletionTimestamp.IsZero() && v1alpha1.WorkflowType(workflowType) == v1alpha1.ReverseWorkflow {
		if workflowpkg.ToConditionReason(workflow.Status.Phase) == meta.SucceededReason ||
			workflowpkg.ToConditionReason(workflow.Status.Phase) == meta.FailedReason {
			// Remove the finalizer because we have finished reversing
			controllerutil.RemoveFinalizer(parent, v1alpha1.AppGroupFinalizer)
			if err := r.Patch(ctx, parent, patch); err != nil {
				return result, err
			}
		}
	}

	// Update the status based on the current state of the helm charts
	// and the status of the workflows
	if err := statusHelper.UpdateStatus(ctx, parent, workflow, v1alpha1.WorkflowType(workflowType)); err != nil {
		logr.Error(err, "failed to update the chart status of the app group")
		return ctrl.Result{}, err
	}
	if err := statusHelper.UpdateFromWorkflowStatus(parent, workflow, v1alpha1.WorkflowType(workflowType)); err != nil {
		logr.Error(err, "failed to update the workflow status of the app group")
		return ctrl.Result{}, err
	}
	if v1alpha1.WorkflowType(workflowType) == v1alpha1.ForwardWorkflow &&
		workflowpkg.ToConditionReason(workflow.Status.Phase) == meta.FailedReason {
		if lastSuccessfulSpec := parent.GetLastSuccessful(); lastSuccessfulSpec != nil {
			if err := reconcileHelper.Rollback(ctx); err != nil {
				logr.Error(err, "failed to generate the rollback workflow")
				return ctrl.Result{}, err
			}
		} else {
			if err := reconcileHelper.Reverse(ctx); err != nil {
				logr.Error(err, "failed to generate the reverse workflow")
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *WorkflowStatusReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha13.Workflow{}).
		Complete(r)
}
