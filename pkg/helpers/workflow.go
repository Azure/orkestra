package helpers

import (
	"context"
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/Orkestra/api/v1alpha1"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
)

func (helper *ReconcileHelper) generateWorkflow(ctx context.Context, logr logr.Logger, g *v1alpha1.ApplicationGroup) error {
	err := helper.Engine.Generate(ctx, logr, g)
	if err != nil {
		logr.Error(err, "engine failed to generate workflow")
		return fmt.Errorf("failed to generate workflow : %w", err)
	}

	err = helper.Engine.Submit(ctx, logr, g)
	if err != nil {
		logr.Error(err, "engine failed to submit workflow")
		return err
	}
	return nil
}

func (helper *ReconcileHelper) generateReverseWorkflow(ctx context.Context, logr logr.Logger, nodes map[string]v1alpha12.NodeStatus, wf *v1alpha12.Workflow) (err error) {
	err = helper.Engine.GenerateReverse(ctx, logr, nodes, wf)
	if err != nil {
		logr.Error(err, "engine failed to generate reverse workflow")
		return fmt.Errorf("failed to generate reverse workflow : %w", err)
	}

	err = helper.Engine.SubmitReverse(ctx, logr, wf)
	if err != nil {
		logr.Error(err, "engine failed to submit reverse workflow")
		return err
	}
	return nil
}

func (helper *ReconcileHelper) cleanupWorkflow(ctx context.Context, logr logr.Logger, workflow *v1alpha12.Workflow) error {
	nodes := make(map[string]v1alpha12.NodeStatus)

	// suspend the forward workflow if it is still running
	err := helper.suspendWorkflow(ctx, logr, workflow)
	if err != nil {
		logr.Error(err, "failed to suspend forward workflow")
		return err
	}
	for _, node := range workflow.Status.Nodes {
		nodes[node.ID] = node
	}
	rwf := &v1alpha12.Workflow{}

	rwfName := fmt.Sprintf("%s-reverse", workflow.Name)
	rwfNamespace := workflow.Namespace
	err = helper.Get(ctx, types.NamespacedName{Namespace: rwfNamespace, Name: rwfName}, rwf)
	if err != nil {
		if kerrors.IsNotFound(err) {
			logr.Info("Reversing the workflow")

			err = helper.generateReverseWorkflow(ctx, logr, nodes, workflow)
			if err != nil {
				logr.Error(err, "failed to generate reverse workflow")
				// if generation of reverse workflow failed, delete the forward workflow and return
				err = helper.Delete(ctx, workflow)
				if err != nil {
					logr.Error(err, "failed to delete workflow CRO")
					return err
				}
				return err
			}
			return nil
		}
		logr.Error(err, "failed to GET workflow object with an unrecoverable error")
	} else {
		// check the completion of the reverse workflow
		if !rwf.Status.FinishedAt.IsZero() {
			logr.Info("reverse workflow is finished")

			err = helper.Delete(ctx, workflow)
			if err != nil {
				logr.Error(err, "failed to delete workflow CRO - continuing with cleanup")
				return err
			}

			return err
		}
		// reverse workflow is not finished - requeue
		return nil
	}
	return nil
}

// suspend a workflow if it is not already finished or suspended
func (helper *ReconcileHelper) suspendWorkflow(ctx context.Context, logr logr.Logger, wf *v1alpha12.Workflow) error {
	if !wf.Status.FinishedAt.IsZero() {
		return nil
	}
	if wf.Spec.Suspend == nil || !*wf.Spec.Suspend {
		wfPatch := client.MergeFrom(wf.DeepCopy())
		suspend := true
		wf.Spec.Suspend = &suspend
		err := helper.Patch(ctx, wf, wfPatch)
		if err != nil {
			logr.Error(err, "failed to patch workflow")
			return err
		}
	}
	return nil
}
