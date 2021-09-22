package workflow

import (
	"context"
	"fmt"
	"github.com/Azure/Orkestra/pkg/graph"
	"github.com/Azure/Orkestra/pkg/templates"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/meta"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
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

	templateGenerator := templates.NewTemplateGenerator(wc.Namespace, wc.Parallelism)
	if err := templateGenerator.GenerateTemplates(combinedGraph); err != nil {
		return fmt.Errorf("failed to generate templates: %w", err)
	}
	templateGenerator.AssignWorkflowTemplates(wc.workflow)
	return nil
}

func (wc *RollbackWorkflowClient) Submit(ctx context.Context) error {
	// Create the new workflow, only if there is not already a rollback workflow that has been created
	wc.workflow.Labels[v1alpha1.OwnershipLabel] = wc.appGroup.Name
	if err := controllerutil.SetControllerReference(wc.appGroup, wc.workflow, wc.Scheme()); err != nil {
		return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
	}
	if err := wc.Create(ctx, wc.workflow); !errors.IsAlreadyExists(err) && err != nil {
		return fmt.Errorf("failed to CREATE argo workflow object: %w", err)
	}
	return nil
}
