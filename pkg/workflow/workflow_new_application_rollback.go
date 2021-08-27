package workflow

import (
	"context"
	"fmt"
	"github.com/Azure/Orkestra/pkg/executor"
	"github.com/Azure/Orkestra/pkg/graph"
	"github.com/Azure/Orkestra/pkg/templates"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (wc *NewApplicationRollbackWorkflowClient) GetLogger() logr.Logger {
	return wc.Logger
}

func (wc *NewApplicationRollbackWorkflowClient) GetClient() client.Client {
	return wc.Client
}

func (wc *NewApplicationRollbackWorkflowClient) GetType() v1alpha1.WorkflowType {
	return v1alpha1.NewApplicationRollback
}

func (wc *NewApplicationRollbackWorkflowClient) GetName() string {
	return fmt.Sprintf("%s-newapp-rollback", wc.appGroup.Name)
}

func (wc *NewApplicationRollbackWorkflowClient) GetNamespace() string {
	return wc.Namespace
}

func (wc *NewApplicationRollbackWorkflowClient) GetOptions() ClientOptions {
	return wc.ClientOptions
}

func (wc *NewApplicationRollbackWorkflowClient) GetAppGroup() *v1alpha1.ApplicationGroup {
	return wc.appGroup
}

func (wc *NewApplicationRollbackWorkflowClient) Generate(ctx context.Context) error {
	if wc.appGroup == nil {
		return fmt.Errorf("applicationGroup object cannot be nil")
	}

	lastAppGroup := wc.GetAppGroup().DeepCopy()
	lastAppGroup.Spec = *wc.appGroup.GetLastSuccessful()
	currGraph := graph.NewForwardGraph(wc.appGroup)
	lastGraph := graph.NewForwardGraph(lastAppGroup)

	diffGraph := graph.GetDiff(currGraph, lastGraph)

	wc.workflow = templates.GenerateWorkflow(wc.GetName(), wc.Namespace, wc.Parallelism)
	entryTemplate, tpls, err := templates.GenerateTemplates(diffGraph.Reverse(), wc.Namespace, wc.Parallelism)
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Update with the app dag templates, entry template, and executor template
	templates.UpdateWorkflowTemplates(wc.workflow, tpls...)
	templates.UpdateWorkflowTemplates(wc.workflow, *entryTemplate, wc.executor(HelmReleaseExecutorName, executor.Install))
	return nil
}

func (wc *NewApplicationRollbackWorkflowClient) Submit(ctx context.Context) error {
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
