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

func (r *ApplicationGroupReconciler) generateWorkflow(ctx context.Context, logr logr.Logger, g *v1alpha1.ApplicationGroup) (requeue bool, err error) {
	err = r.Engine.Generate(ctx, logr, g)
	if err != nil {
		logr.Error(err, "engine failed to generate workflow")
		return false, fmt.Errorf("failed to generate workflow : %w", err)
	}

	err = r.Engine.Submit(ctx, logr, g)
	if err != nil {
		logr.Error(err, "engine failed to submit workflow")
		return false, err
	}
	return true, nil
}

func (r *ApplicationGroupReconciler) generateReverseWorkflow(ctx context.Context, logr logr.Logger, nodes map[string]v1alpha12.NodeStatus, wf *v1alpha12.Workflow) (err error) {
	err = r.Engine.GenerateReverse(ctx, logr, nodes, wf)
	if err != nil {
		logr.Error(err, "engine failed to generate reverse workflow")
		return fmt.Errorf("failed to generate reverse workflow : %w", err)
	}

	err = r.Engine.SubmitReverse(ctx, logr, wf)
	if err != nil {
		logr.Error(err, "engine failed to submit reverse workflow")
		return err
	}
	return nil
}

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
		for _, node := range wf.Status.Nodes {
			nodes[node.ID] = node
		}
		rwf := &v1alpha12.Workflow{}

		rwfName := fmt.Sprintf("%s-reverse", wf.Name)
		rwfNamespace := wf.Namespace
		err := r.Client.Get(ctx, types.NamespacedName{Namespace: rwfNamespace, Name: rwfName}, rwf)
		if err != nil {
			if kerrors.IsNotFound(err) {
				logr.Info("Reversing the workflow")

				err = r.generateReverseWorkflow(ctx, logr, nodes, &wf)
				if err != nil {
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
