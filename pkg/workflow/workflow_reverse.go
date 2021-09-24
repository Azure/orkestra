package workflow

import (
	"context"
	"fmt"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/graph"
	"github.com/Azure/Orkestra/pkg/meta"
	"github.com/Azure/Orkestra/pkg/templates"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

func (wc *ReverseWorkflowClient) GetWorkflow() *v1alpha13.Workflow {
	return wc.workflow
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
	forwardClient := NewClientFromClient(wc, v1alpha1.Forward)
	rollbackClient := NewClientFromClient(wc, v1alpha1.Rollback)

	if err := Suspend(ctx, forwardClient); err != nil {
		return fmt.Errorf("failed to suspend forward workflow: %w", err)
	}
	if err := Suspend(ctx, rollbackClient); err != nil {
		return fmt.Errorf("failed to suspend rollback workflow: %w", err)
	}

	wc.workflow = templates.GenerateWorkflow(wc.GetName(), wc.Namespace, wc.Parallelism)
	graph := graph.NewReverseGraph(wc.GetAppGroup())

	templateGenerator := templates.NewTemplateGenerator(wc.Namespace, wc.Parallelism)
	if err := templateGenerator.GenerateTemplates(graph); err != nil {
		return fmt.Errorf("failed to generate templates: %w", err)
	}
	templateGenerator.AssignWorkflowTemplates(wc.workflow)
	return nil
}

func (wc *ReverseWorkflowClient) Submit(ctx context.Context) error {
	forwardClient := NewClientFromClient(wc, v1alpha1.Forward)
	forwardWorkflow, err := GetWorkflow(ctx, forwardClient)
	if errors.IsNotFound(err) {
		return meta.ErrForwardWorkflowNotFound
	} else if err != nil {
		return err
	}
	wc.workflow.Labels[v1alpha1.WorkflowTypeLabel] = string(v1alpha1.Reverse)
	if err := controllerutil.SetControllerReference(forwardWorkflow, wc.workflow, wc.Scheme()); err != nil {
		return fmt.Errorf("unable to set forward workflow as owner of Argo reverse Workflow: %w", err)
	}
	if err := Submit(ctx, wc); err != nil {
		return err
	}
	return nil
}
