package workflow

import (
	"context"
	"fmt"

	"github.com/Azure/Orkestra/pkg/graph"
	"github.com/Azure/Orkestra/pkg/templates"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (wc *ForwardWorkflowClient) GetLogger() logr.Logger {
	return wc.Logger
}

func (wc *ForwardWorkflowClient) GetClient() client.Client {
	return wc.Client
}

func (wc *ForwardWorkflowClient) GetType() v1alpha1.WorkflowType {
	return v1alpha1.Forward
}

func (wc *ForwardWorkflowClient) GetName() string {
	return wc.appGroup.Name
}

func (wc *ForwardWorkflowClient) GetNamespace() string {
	return wc.Namespace
}

func (wc *ForwardWorkflowClient) GetOptions() ClientOptions {
	return wc.ClientOptions
}

func (wc *ForwardWorkflowClient) GetAppGroup() *v1alpha1.ApplicationGroup {
	return wc.appGroup
}

func (wc *ForwardWorkflowClient) Generate(ctx context.Context) error {
	if wc.appGroup == nil {
		return fmt.Errorf("applicationGroup object cannot be nil")
	}

	// Suspend the rollback or reverse workflows if they are running
	reverseClient := NewBuilderFromClient(wc).Reverse(wc.appGroup).Build()
	rollbackClient := NewBuilderFromClient(wc).Rollback(wc.appGroup).Build()
	if err := Suspend(ctx, reverseClient); err != nil {
		return fmt.Errorf("failed to suspend reverse workflow: %w", err)
	}
	if err := Suspend(ctx, rollbackClient); err != nil {
		return fmt.Errorf("failed to suspend rollback workflow: %w", err)
	}

	wc.workflow = templates.GenerateWorkflow(wc.appGroup.Name, wc.Namespace, wc.Parallelism)
	graph := graph.NewForwardGraph(wc.GetAppGroup())
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

func (wc *ForwardWorkflowClient) Submit(ctx context.Context) error {
	if wc.workflow == nil {
		return fmt.Errorf("workflow object cannot be nil")
	}
	if wc.appGroup == nil {
		return fmt.Errorf("applicationGroup object cannot be nil")
	}

	// Create the Workflow
	wc.workflow.Labels[v1alpha1.OwnershipLabel] = wc.appGroup.Name
	if err := controllerutil.SetControllerReference(wc.appGroup, wc.workflow, wc.Scheme()); err != nil {
		return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
	}
	if err := wc.Create(ctx, wc.workflow); !errors.IsAlreadyExists(err) && err != nil {
		return fmt.Errorf("failed to CREATE argo workflow object: %w", err)
	} else if errors.IsAlreadyExists(err) {
		// If the workflow needs an update, delete the previous workflow and apply the new one
		// Argo Workflow does not rerun the workflow on UPDATE, so intead we cleanup and reapply
		if err := wc.Delete(ctx, wc.workflow); err != nil {
			return fmt.Errorf("failed to DELETE argo workflow object: %w", err)
		}
		if err := controllerutil.SetControllerReference(wc.appGroup, wc.workflow, wc.Scheme()); err != nil {
			return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
		}
		// If the argo Workflow object is NotFound and not AlreadyExists on the cluster
		// create a new object and submit it to the cluster
		if err := wc.Create(ctx, wc.workflow); err != nil {
			return fmt.Errorf("failed to CREATE argo workflow object: %w", err)
		}
	}
	return nil
}
