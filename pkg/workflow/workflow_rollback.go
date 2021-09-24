package workflow

import (
	"context"
	"fmt"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/Azure/Orkestra/pkg/graph"
	"github.com/Azure/Orkestra/pkg/templates"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/meta"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (wc *RollbackWorkflowClient) GetLogger() logr.Logger {
	return wc.Logger
}

func (wc *RollbackWorkflowClient) GetClient() client.Client {
	return wc.Client
}

func (wc *RollbackWorkflowClient) GetType() v1alpha1.WorkflowType {
	return v1alpha1.Rollback
}

func (wc *RollbackWorkflowClient) GetName() string {
	return fmt.Sprintf("%s-rollback", wc.appGroup.Name)
}

func (wc *RollbackWorkflowClient) GetNamespace() string {
	return wc.Namespace
}

func (wc *RollbackWorkflowClient) GetOptions() ClientOptions {
	return wc.ClientOptions
}

func (wc *RollbackWorkflowClient) GetAppGroup() *v1alpha1.ApplicationGroup {
	return wc.appGroup
}

func (wc *RollbackWorkflowClient) GetWorkflow() *v1alpha13.Workflow {
	return wc.workflow
}

func (wc *RollbackWorkflowClient) Generate(ctx context.Context) error {
	if wc.appGroup == nil {
		return fmt.Errorf("applicationGroup object cannot be nil")
	}

	rollbackAppGroup := wc.appGroup.DeepCopy()
	lastSuccessful := wc.appGroup.GetLastSuccessful()
	if lastSuccessful == nil {
		return meta.ErrPreviousSpecNotSet
	}
	rollbackAppGroup.Spec = *lastSuccessful

	currGraph := graph.NewForwardGraph(wc.appGroup)
	lastGraph := graph.NewForwardGraph(rollbackAppGroup)
	diffGraph := graph.Diff(currGraph, lastGraph)

	wc.workflow = templates.GenerateWorkflow(wc.GetName(), wc.Namespace, wc.Parallelism)
	combinedGraph := graph.Combine(lastGraph, diffGraph.Reverse())
	entryTemplate, tpls, err := templates.GenerateTemplates(combinedGraph, wc.Namespace, wc.Parallelism)
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Update with the app dag templates, entry template, and executor template
	templates.UpdateWorkflowTemplates(wc.workflow, tpls...)
	templates.UpdateWorkflowTemplates(wc.workflow, *entryTemplate)
	for _, executor := range combinedGraph.AllExecutors {
		templates.UpdateWorkflowTemplates(wc.workflow, executor.GetTemplate())
	}
	return nil
}

func (wc *RollbackWorkflowClient) Submit(ctx context.Context) error {
	wc.workflow.Labels[v1alpha1.WorkflowTypeLabel] = string(v1alpha1.RollbackWorkflow)
	if err := controllerutil.SetControllerReference(wc.appGroup, wc.workflow, wc.Scheme()); err != nil {
		return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
	}
	if err := Submit(ctx, wc); err != nil {
		return err
	}
	return nil
}
