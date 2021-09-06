package workflow

import (
	"context"
	"fmt"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/Azure/Orkestra/pkg/graph"
	"github.com/Azure/Orkestra/pkg/templates"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (wc *ForwardWorkflowClient) GetWorkflow() *v1alpha13.Workflow {
	return wc.workflow
}

func (wc *ForwardWorkflowClient) Generate(ctx context.Context) error {
	if wc.appGroup == nil {
		return fmt.Errorf("applicationGroup object cannot be nil")
	}

	// Suspend the rollback or reverse workflows if they are running
	reverseClient := NewClientFromClient(wc, v1alpha1.ReverseWorkflow)
	rollbackClient := NewClientFromClient(wc, v1alpha1.RollbackWorkflow)
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
	if err := wc.createTargetNamespaces(ctx); err != nil {
		return fmt.Errorf("failed to create the target namespaces: %w", err)
	}
	wc.workflow.Labels[v1alpha1.WorkflowTypeLabel] = string(v1alpha1.ForwardWorkflow)
	if err := controllerutil.SetControllerReference(wc.appGroup, wc.workflow, wc.Scheme()); err != nil {
		return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
	}
	if err := Submit(ctx, wc); err != nil {
		return err
	}
	return nil
}

func (wc *ForwardWorkflowClient) createTargetNamespaces(ctx context.Context) error {
	namespaces := []string{}
	// Add namespaces we need to create while removing duplicates
	for _, app := range wc.appGroup.Spec.Applications {
		found := false
		for _, namespace := range namespaces {
			if app.Spec.Release.TargetNamespace == namespace {
				found = true
				break
			}
		}
		if !found {
			namespaces = append(namespaces, app.Spec.Release.TargetNamespace)
		}
	}

	// Create any of the target namespaces
	for _, namespace := range namespaces {
		ns := &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: namespace,
				Labels: map[string]string{
					"name": namespace,
				},
			},
		}
		if err := controllerutil.SetControllerReference(wc.appGroup, ns, wc.Scheme()); err != nil {
			return fmt.Errorf("failed to set OwnerReference for Namespace %s: %w", ns.Name, err)
		}
		if err := wc.Create(ctx, ns); !errors.IsAlreadyExists(err) && err != nil {
			return fmt.Errorf("failed to CREATE namespace %s object: %w", ns.Name, err)
		}
	}
	return nil
}
