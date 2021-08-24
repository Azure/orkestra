package workflow

import (
	"context"
	"fmt"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
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

func (wc *ReverseWorkflowClient) GetNamespace() string {
	return wc.namespace
}

func (wc *ReverseWorkflowClient) GetWorkflow(ctx context.Context) (*v1alpha13.Workflow, error) {
	reverseWorkflow := &v1alpha13.Workflow{}

	rwfName := fmt.Sprintf("%s-reverse", wc.appGroup.Name)
	err := wc.Get(ctx, types.NamespacedName{Namespace: wc.namespace, Name: rwfName}, reverseWorkflow)
	return reverseWorkflow, err
}

func (wc *ReverseWorkflowClient) Generate(ctx context.Context) error {
	var err error

	forwardClient := NewBuilderFromClient(wc).Forward(wc.appGroup).Build()

	// Get the forward workflow from the server and suspend it if it's still running
	wc.forwardWorkflow, err = forwardClient.GetWorkflow(ctx)
	if client.IgnoreNotFound(err) != nil {
		return err
	} else if err != nil {
		return meta.ErrForwardWorkflowNotFound
	}
	if err := Suspend(ctx, forwardClient); err != nil {
		return fmt.Errorf("failed to suspend forward workflow: %w", err)
	}

	wc.reverseWorkflow = initWorkflowObject(wc.getReverseName(), wc.namespace, wc.parallelism)
	graph := NewReverseGraph(wc.GetAppGroup())
	entryTemplate, templates, err := generateTemplates(graph, wc.GetOptions())
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Update with the app dag templates, entry template, and executor template
	updateWorkflowTemplates(wc.reverseWorkflow, templates...)
	updateWorkflowTemplates(wc.reverseWorkflow, *entryTemplate, wc.executor(HelmReleaseExecutorName, Delete))

	return nil
}

func (wc *ReverseWorkflowClient) Submit(ctx context.Context) error {
	if wc.forwardWorkflow == nil {
		wc.Error(nil, "forward workflow object cannot be nil")
		return fmt.Errorf("forward workflow object cannot be nil")
	}
	obj := &v1alpha13.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      wc.reverseWorkflow.Name,
			Namespace: wc.reverseWorkflow.Namespace,
		},
	}
	if err := wc.Get(ctx, client.ObjectKeyFromObject(obj), obj); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to GET workflow object with an unrecoverable error: %w", err)
	} else if err != nil {
		if err := controllerutil.SetControllerReference(wc.forwardWorkflow, wc.reverseWorkflow, wc.Scheme()); err != nil {
			return fmt.Errorf("unable to set forward workflow as owner of Argo reverse Workflow: %w", err)
		}
		// If the argo Workflow object is NotFound and not AlreadyExists on the cluster
		// create a new object and submit it to the cluster
		if err = wc.Create(ctx, wc.reverseWorkflow); err != nil {
			return fmt.Errorf("failed to CREATE argo workflow object: %w", err)
		}
	}
	return nil
}


func (wc *ReverseWorkflowClient) getReverseName() string {
	return fmt.Sprintf("%s-reverse", wc.forwardWorkflow.Name)
}
