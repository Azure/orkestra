package workflow

import (
	"context"
	"fmt"
	"github.com/Azure/Orkestra/api/v1alpha1"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (wc *RollbackWorkflowClient) GetLogger() logr.Logger {
	return wc.Logger
}

func (wc *RollbackWorkflowClient) GetClient() client.Client {
	return wc.Client
}

func (wc *RollbackWorkflowClient) GetType() ClientType {
	return Rollback
}

func (wc *RollbackWorkflowClient) GetNamespace() string {
	return wc.namespace
}

func (wc *RollbackWorkflowClient) GetOptions() ClientOptions {
	return wc.ClientOptions
}

func (wc *RollbackWorkflowClient) GetAppGroup() *v1alpha1.ApplicationGroup {
	return wc.appGroup
}

func (wc *RollbackWorkflowClient) GetWorkflow(ctx context.Context) (*v1alpha12.Workflow, error) {
	rollbackWorkflow := &v1alpha12.Workflow{}
	rollbackWorkflowName := fmt.Sprintf("%s-rollback", wc.appGroup.Name)
	err := wc.Get(ctx, types.NamespacedName{Namespace: wc.namespace, Name: rollbackWorkflowName}, rollbackWorkflow)
	return rollbackWorkflow, err
}

func (wc *RollbackWorkflowClient) Generate(ctx context.Context) error {
	if wc.appGroup == nil {
		wc.Error(nil, "ApplicationGroup object cannot be nil")
		return fmt.Errorf("applicationGroup object cannot be nil")
	}

	rollbackWorkflowName := fmt.Sprintf("%s-rollback", wc.appGroup.Name)
	wc.workflow = initWorkflowObject(rollbackWorkflowName, wc.namespace, wc.parallelism)

	entryTemplate, templates, err := generateTemplates(wc)
	if err != nil {
		wc.Error(err, "failed to generate workflow")
		return fmt.Errorf("failed to generate argo workflow : %w", err)
	}
	// Update with the app dag templates, entry template, and executor template
	updateWorkflowTemplates(wc.workflow, templates...)
	updateWorkflowTemplates(wc.workflow, *entryTemplate, wc.executor(HelmReleaseExecutorName, Install))

	return nil
}

func (wc *RollbackWorkflowClient) Submit(ctx context.Context) error {
	// Create the Workflow
	wc.workflow.Labels[OwnershipLabel] = wc.appGroup.Name
	if err := controllerutil.SetControllerReference(wc.appGroup, wc.workflow, wc.Scheme()); err != nil {
		wc.Error(err, "unable to set ApplicationGroup as owner of Argo Workflow object")
		return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
	}
	if err := wc.Create(ctx, wc.workflow); !errors.IsAlreadyExists(err) && err != nil {
		wc.Error(err, "failed to CREATE argo workflow object")
		return fmt.Errorf("failed to CREATE argo workflow object : %w", err)
	} else if errors.IsAlreadyExists(err) {
		if err := wc.Delete(ctx, wc.workflow); err != nil {
			wc.Error(err, "failed to DELETE argo workflow object")
			return fmt.Errorf("failed to DELETE argo workflow object : %w", err)
		}
		if err := controllerutil.SetControllerReference(wc.appGroup, wc.workflow, wc.Scheme()); err != nil {
			wc.Error(err, "unable to set ApplicationGroup as owner of Argo Workflow object")
			return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
		}
		if err := wc.Create(ctx, wc.workflow); err != nil {
			wc.Error(err, "failed to CREATE argo workflow object")
			return fmt.Errorf("failed to CREATE argo workflow object : %w", err)
		}
	}
	return nil
}
