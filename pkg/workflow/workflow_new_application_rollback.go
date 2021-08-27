package workflow

import (
	"context"
	"fmt"
	"github.com/Azure/Orkestra/pkg/executor"
	"github.com/Azure/Orkestra/pkg/graph"
	"github.com/Azure/Orkestra/pkg/templates"

	"github.com/Azure/Orkestra/api/v1alpha1"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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

func (wc *NewApplicationRollbackWorkflowClient) GetWorkflow(ctx context.Context) (*v1alpha13.Workflow, error) {
	rollbackWorkflow := &v1alpha13.Workflow{}
	rollbackWorkflowName := fmt.Sprintf("%s-rollback", wc.appGroup.Name)
	err := wc.Get(ctx, types.NamespacedName{Namespace: wc.Namespace, Name: rollbackWorkflowName}, rollbackWorkflow)
	return rollbackWorkflow, err
}

func (wc *NewApplicationRollbackWorkflowClient) Generate(ctx context.Context) error {
	if wc.appGroup == nil {
		return fmt.Errorf("applicationGroup object cannot be nil")
	}

	wc.workflow = initWorkflowObject(wc.GetName(), wc.Namespace, wc.Parallelism)
	graph := graph.NewForwardGraph(wc.appGroup)
	entryTemplate, templates, err := templates.GenerateTemplates(graph, wc.GetOptions())
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Update with the app dag templates, entry template, and executor template
	updateWorkflowTemplates(wc.workflow, templates...)
	updateWorkflowTemplates(wc.workflow, *entryTemplate, wc.executor(HelmReleaseExecutorName, executor.Install))
	return nil
}

func (wc *NewApplicationRollbackWorkflowClient) Submit(ctx context.Context) error {
	if err := wc.purgeNewerReleases(ctx); err != nil {
		return fmt.Errorf("failed to purge helm releases: %w", err)
	}

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