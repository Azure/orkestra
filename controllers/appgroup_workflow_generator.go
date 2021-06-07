package controllers

import (
	"context"
	"fmt"

	"github.com/Azure/Orkestra/pkg/workflow"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/Orkestra/api/v1alpha1"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
)

func (r *ApplicationGroupReconciler) cleanupWorkflow(ctx context.Context, logr logr.Logger, g *v1alpha1.ApplicationGroup) bool {
	nodes := make(map[string]v1alpha12.NodeStatus)
	wfs := v1alpha12.WorkflowList{}
	listOption := client.MatchingLabels{
		workflow.OwnershipLabel: g.Name,
		workflow.HeritageLabel:  workflow.Project,
	}
	_ = r.List(ctx, &wfs, listOption)

	if wfs.Items.Len() != 0 {
		wf := wfs.Items[0]
		// suspend the forward workflow if it is still running
		err := r.suspendWorkflow(ctx, logr, &wf)
		if err != nil {
			logr.Error(err, "failed to suspend forward workflow")
			return false
		}
		for _, node := range wf.Status.Nodes {
			nodes[node.ID] = node
		}
		rwf := &v1alpha12.Workflow{}

		rwfName := fmt.Sprintf("%s-reverse", wf.Name)
		rwfNamespace := wf.Namespace
		err = r.Client.Get(ctx, types.NamespacedName{Namespace: rwfNamespace, Name: rwfName}, rwf)
		if err != nil {
			if kerrors.IsNotFound(err) {
				logr.Info("Reversing the workflow")

				engine, _ := r.EngineBuilder.Reverse(&wf, nodes).Build()
				if err := workflow.Run(ctx, engine); err != nil {
					logr.Error(err, "failed to generate reverse workflow")
					// if generation of reverse workflow failed, delete the forward workflow and return
					err = r.Client.Delete(ctx, &wf)
					if err != nil {
						logr.Error(err, "failed to delete workflow CRO")
						return false
					}
					return false
				}

				// reverse workflow started - requeue
				return true
			}
			logr.Error(err, "failed to GET workflow object with an unrecoverable error")
		} else {
			// check the completion of the reverse workflow
			if !rwf.Status.FinishedAt.IsZero() {
				logr.Info("reverse workflow is finished")

				err = r.Client.Delete(ctx, &wf)
				if err != nil {
					logr.Error(err, "failed to delete workflow CRO - continuing with cleanup")
					return false
				}

				return false
			}
			// reverse workflow is not finished - requeue
			return true
		}
	}
	return false
}

// suspend a workflow if it is not already finished or suspended
func (r *ApplicationGroupReconciler) suspendWorkflow(ctx context.Context, logr logr.Logger, wf *v1alpha12.Workflow) error {
	if !wf.Status.FinishedAt.IsZero() {
		return nil
	}
	if wf.Spec.Suspend == nil || !*wf.Spec.Suspend {
		wfPatch := client.MergeFrom(wf.DeepCopy())
		suspend := true
		wf.Spec.Suspend = &suspend
		err := r.Client.Patch(ctx, wf, wfPatch)
		if err != nil {
			logr.Error(err, "failed to patch workflow")
			return err
		}
	}
	return nil
}
