// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Azure/Orkestra/pkg/meta"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

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

func (r *WorkflowStatusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	workflow := &v1alpha13.Workflow{}
	logr := r.Log.WithValues("workflow_name", req.NamespacedName.Name)

	if err := r.Get(ctx, req.NamespacedName, workflow); err != nil {
		if errors.IsNotFound(err) {
			logr.V(3).Info("workflow not found in the cluster")
			return ctrl.Result{}, nil
		}
		logr.Error(err, "failed to fetch workflow instance")
		return ctrl.Result{}, err
	}
	if !workflow.DeletionTimestamp.IsZero() {
		workflowPatch := client.MergeFrom(workflow.DeepCopy())
		logr.V(2).Info("got delete event for the workflow")
		controllerutil.RemoveFinalizer(workflow, v1alpha1.AppGroupFinalizer)
		if err := r.Patch(ctx, workflow, workflowPatch); err != nil {
			logr.Error(err, "failed to remove the finalizer from the workflow object on deletion")
			return ctrl.Result{}, err
		}

		// If the parent is in a deleting state, then we need to check if this workflow deletion is the
		// reverse workflow for this application group. If it is, then we remove the finalizer
		parent, workflowType, err := r.getParentAndWorkflowType(ctx, workflow)
		if err != nil {
			if errors.IsNotFound(err) {
				logr.V(3).Info("parent application group not found in the cluster")
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}
		logr = logr.WithValues("workflow_type", workflowType, "parent", parent.Name)
		if !parent.DeletionTimestamp.IsZero() && workflowType == v1alpha1.ReverseWorkflow {
			patch := client.MergeFrom(parent.DeepCopy())
			logr.V(2).Info("removing the finalizer from the parent due to us losing the reverse workflow")
			controllerutil.RemoveFinalizer(parent, v1alpha1.AppGroupFinalizer)
			if err := r.Patch(ctx, parent, patch); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	parent, workflowType, err := r.getParentAndWorkflowType(ctx, workflow)
	if err != nil {
		if errors.IsNotFound(err) {
			logr.V(3).Info("parent application group not found in the cluster")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	logr = logr.WithValues("workflow_type", workflowType, "parent", parent.Name)

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

	workflowAppGeneration, err := getWorkflowAppGeneration(workflow)
	if err != nil {
		logr.Error(err, "failed to get the workflow app generation")
		return ctrl.Result{}, nil
	}

	// If the parent generation doesn't match this workflow's generation
	// we can ignore updates to this workflow
	if parent.Generation != workflowAppGeneration {
		logr.V(3).Info("workflow generation does not match parent generation")
		return ctrl.Result{}, nil
	}

	if parent.Generation != parent.Status.ObservedGeneration {
		logr.V(3).Info("parent is still reconciling")
		return ctrl.Result{Requeue: true}, nil
	}

	// Remove the finalizer from the parent application group if we have just completed reversing
	if !parent.DeletionTimestamp.IsZero() && workflowType == v1alpha1.ReverseWorkflow &&
		(workflowpkg.ToConditionReason(workflow.Status.Phase) == meta.SucceededReason || workflowpkg.ToConditionReason(workflow.Status.Phase) == meta.FailedReason) {
		// Remove the finalizer because we have finished reversing
		controllerutil.RemoveFinalizer(parent, v1alpha1.AppGroupFinalizer)
		if err := r.Patch(ctx, parent, patch); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Update the status based on the current state of the helm charts
	// and the status of the workflows
	if err := statusHelper.UpdateStatus(ctx, parent, workflow, workflowType); err != nil {
		logr.Error(err, "failed to update the chart status of the app group")
		return ctrl.Result{}, err
	}
	if err := statusHelper.UpdateFromWorkflowStatus(parent, workflow, workflowType); err != nil {
		logr.Error(err, "failed to update the workflow status of the app group")
		return ctrl.Result{}, err
	}
	if err := statusHelper.PatchStatus(ctx, parent); err != nil {
		logr.Error(err, "failed to patch the status of the parent based on the workflow")
		return ctrl.Result{}, err
	}
	if workflowType == v1alpha1.ForwardWorkflow &&
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

func getWorkflowAppGeneration(workflow *v1alpha13.Workflow) (int64, error) {
	workflowAppGroupGenerationStr, ok := workflow.GetLabels()[v1alpha1.WorkflowAppGroupGenerationLabel]
	if !ok {
		return -1, fmt.Errorf("workflow app gorup generation not found for the child workflow")
	}
	workflowAppGroupGeneration, err := strconv.ParseInt(workflowAppGroupGenerationStr, 10, 64)
	if err != nil {
		return -1, fmt.Errorf("workflow app gorup generation not able to be parsed for child workflow")
	}
	return workflowAppGroupGeneration, nil
}

func orkestraOwnedPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			workflow := e.Object.(*v1alpha13.Workflow)
			return hasValidOrkestraLabels(workflow)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			workflow := e.ObjectNew.(*v1alpha13.Workflow)
			return hasValidOrkestraLabels(workflow)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			workflow := e.Object.(*v1alpha13.Workflow)
			return hasValidOrkestraLabels(workflow)
		},
	}
}

func hasValidOrkestraLabels(workflow *v1alpha13.Workflow) bool {
	return workflow.GetLabels()[v1alpha1.OwnershipLabel] != "" &&
		workflow.GetLabels()[v1alpha1.WorkflowTypeLabel] != "" &&
		workflow.GetLabels()[v1alpha1.WorkflowAppGroupGenerationLabel] != ""
}

func (r *WorkflowStatusReconciler) getParentApplicationGroup(ctx context.Context, workflow *v1alpha13.Workflow) (*v1alpha1.ApplicationGroup, error) {
	parent := &v1alpha1.ApplicationGroup{}
	if appGroupName, ok := workflow.GetLabels()[v1alpha1.OwnershipLabel]; ok {
		if err := r.Get(ctx, types.NamespacedName{Name: appGroupName}, parent); err != nil {
			return nil, err
		}
	}
	return parent, nil
}

func (r *WorkflowStatusReconciler) getParentAndWorkflowType(ctx context.Context, workflow *v1alpha13.Workflow) (*v1alpha1.ApplicationGroup, v1alpha1.WorkflowType, error) {
	parent := &v1alpha1.ApplicationGroup{}
	appGroupName, ok := workflow.GetLabels()[v1alpha1.OwnershipLabel]
	if !ok {
		err := fmt.Errorf("ownership label not found for the child workflow")
		return nil, "", err
	}
	if err := r.Get(ctx, types.NamespacedName{Name: appGroupName}, parent); err != nil {
		return nil, "", err
	}
	workflowTypeStr, ok := workflow.GetLabels()[v1alpha1.WorkflowTypeLabel]
	workflowType := v1alpha1.WorkflowType(workflowTypeStr)
	if !ok {
		err := fmt.Errorf("workflow type label not found for the child workflow")
		return nil, "", err
	}
	return parent, workflowType, nil
}

func (r *WorkflowStatusReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha13.Workflow{}).
		WithEventFilter(orkestraOwnedPredicate()).
		Complete(r)
}
