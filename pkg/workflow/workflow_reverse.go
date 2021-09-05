package workflow

import (
	"context"
	"fmt"

	"github.com/Azure/Orkestra/pkg/graph"
	"github.com/Azure/Orkestra/pkg/templates"

	"github.com/Azure/Orkestra/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (wc *ReverseWorkflowClient) GetLogger() logr.Logger {
	return wc.Logger
}

func (wc *ReverseWorkflowClient) GetClient() client.Client {
	return wc.Client
}

func (wc *ReverseWorkflowClient) GetType() v1alpha1.WorkflowType {
	return v1alpha1.Rollback
}

func (wc *ReverseWorkflowClient) GetAppGroup() *v1alpha1.ApplicationGroup {
	return wc.appGroup
}

func (wc *ReverseWorkflowClient) GetOptions() ClientOptions {
	return wc.ClientOptions
}

func (wc *ReverseWorkflowClient) GetName() string {
	return fmt.Sprintf("%s-reverse", wc.appGroup.Name)
}

func (wc *ReverseWorkflowClient) GetNamespace() string {
	return wc.Namespace
}

func (wc *ReverseWorkflowClient) Generate(ctx context.Context) error {
	var err error

	forwardClient := NewClientFromClient(wc, v1alpha1.ForwardWorkflow)
	rollbackClient := NewClientFromClient(wc, v1alpha1.RollbackWorkflow)

	if err := Suspend(ctx, forwardClient); err != nil {
		return fmt.Errorf("failed to suspend forward workflow: %w", err)
	}
	if err := Suspend(ctx, rollbackClient); err != nil {
		return fmt.Errorf("failed to suspend rollback workflow: %w", err)
	}

	wc.workflow = templates.GenerateWorkflow(wc.GetName(), wc.Namespace, wc.Parallelism)
	graph := graph.NewReverseGraph(wc.GetAppGroup())
	entryTemplate, tpls, err := templates.GenerateTemplates(graph, wc.Namespace, wc.Parallelism)
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Update with the app dag templates, entry template, and executor template
	templates.UpdateWorkflowTemplates(wc.workflow, tpls...)
	templates.UpdateWorkflowTemplates(wc.workflow, *entryTemplate)
	for _, executor := range graph.AllExecutors {
		templates.UpdateWorkflowTemplates(wc.workflow, executor.GetTemplate())
	}
	return nil
}

func (wc *ReverseWorkflowClient) Submit(ctx context.Context) error {
	forwardClient := NewClientFromClient(wc, v1alpha1.ForwardWorkflow)
	forwardWorkflow, err := GetWorkflow(ctx, forwardClient)
	if err != nil {
		return err
	}
	obj := &v1alpha13.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      wc.workflow.Name,
			Namespace: wc.workflow.Namespace,
		},
	}
	wc.workflow.Labels[v1alpha1.OwnershipLabel] = wc.appGroup.Name
	wc.workflow.Labels[v1alpha1.WorkflowTypeLabel] = string(v1alpha1.ReverseWorkflow)
	if err := wc.Get(ctx, client.ObjectKeyFromObject(obj), obj); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to GET workflow object with an unrecoverable error: %w", err)
	} else if err != nil {
		if err := controllerutil.SetControllerReference(forwardWorkflow, wc.workflow, wc.Scheme()); err != nil {
			return fmt.Errorf("unable to set forward workflow as owner of Argo reverse Workflow: %w", err)
		}
		// If the argo Workflow object is NotFound and not AlreadyExists on the cluster
		// create a new object and submit it to the cluster
		if err = wc.Create(ctx, wc.workflow); err != nil {
			return fmt.Errorf("failed to CREATE argo workflow object: %w", err)
		}
	}
	return nil
}
