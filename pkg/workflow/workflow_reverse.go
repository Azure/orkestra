package workflow

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
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

func (wc *ReverseWorkflowClient) GetType() ClientType {
	return Reverse
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

func (wc *ReverseWorkflowClient) GetWorkflow(ctx context.Context) (*v1alpha12.Workflow, error) {
	reverseWorkflow := &v1alpha12.Workflow{}

	rwfName := fmt.Sprintf("%s-reverse", wc.appGroup.Name)
	err := wc.Get(ctx, types.NamespacedName{Namespace: wc.namespace, Name: rwfName}, reverseWorkflow)
	return reverseWorkflow, err
}

func (wc *ReverseWorkflowClient) Generate(ctx context.Context) error {
	var err error

	// Get the forward workflow from the server and suspend it if it's still running
	wc.forwardWorkflow, err = wc.forwardClient.GetWorkflow(ctx)
	if client.IgnoreNotFound(err) != nil {
		return err
	} else if err != nil {
		return meta.ForwardWorkflowNotFound
	}
	if err := Suspend(ctx, wc.forwardClient); err != nil {
		wc.Error(err, "failed to suspend forward workflow")
		return err
	}

	wc.reverseWorkflow = initWorkflowObject(wc.getReverseName(), wc.namespace, wc.parallelism)

	entry, err := wc.generateWorkflow()
	if err != nil {
		wc.Error(err, "failed to generate reverse workflow")
		return fmt.Errorf("failed to generate argo reverse workflow : %w", err)
	}

	updateWorkflowTemplates(wc.reverseWorkflow, *entry, wc.executor(HelmReleaseReverseExecutorName, Delete))
	return nil
}

func (wc *ReverseWorkflowClient) Submit(ctx context.Context) error {
	if err := wc.validate(); err != nil {
		return err
	}
	obj := &v1alpha12.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      wc.reverseWorkflow.Name,
			Namespace: wc.reverseWorkflow.Namespace,
		},
	}
	if err := wc.Get(ctx, client.ObjectKeyFromObject(obj), obj); client.IgnoreNotFound(err) != nil {
		wc.Error(err, "failed to GET workflow object with an unrecoverable error")
		return fmt.Errorf("failed to GET workflow object with an unrecoverable error : %w", err)
	} else if err != nil {
		if err := controllerutil.SetControllerReference(wc.forwardWorkflow, wc.reverseWorkflow, wc.Scheme()); err != nil {
			wc.Error(err, "unable to set forward workflow as owner of Argo reverse Workflow object")
			return fmt.Errorf("unable to set forward workflow as owner of Argo reverse Workflow: %w", err)
		}
		// If the argo Workflow object is NotFound and not AlreadyExists on the cluster
		// create a new object and submit it to the cluster
		if err = wc.Create(ctx, wc.reverseWorkflow); err != nil {
			wc.Error(err, "failed to CREATE argo workflow object")
			return fmt.Errorf("failed to CREATE argo workflow object : %w", err)
		}
	}
	return nil
}

func (wc *ReverseWorkflowClient) generateWorkflow() (*v1alpha12.Template, error) {
	graph, err := Build(wc.forwardWorkflow.Name, getNodes(wc.forwardWorkflow))
	if err != nil {
		wc.Error(err, "failed to build the wf status DAG")
		return nil, fmt.Errorf("failed to build the wf status DAG : %w", err)
	}

	rev := graph.Reverse()

	entry := &v1alpha12.Template{
		Name: EntrypointTemplateName,
		DAG: &v1alpha12.DAGTemplate{
			Tasks: make([]v1alpha12.DAGTask, 0),
		},
	}

	var prevbucket []fluxhelmv2beta1.HelmRelease
	for _, bucket := range rev {
		for _, hr := range bucket {
			task := v1alpha12.DAGTask{
				Name:     utils.ConvertToDNS1123(hr.GetReleaseName() + "-" + hr.Namespace),
				Template: HelmReleaseReverseExecutorName,
				Arguments: v1alpha12.Arguments{
					Parameters: []v1alpha12.Parameter{
						{
							Name:  HelmReleaseArg,
							Value: utils.ToStrPtr(base64.StdEncoding.EncodeToString([]byte(utils.HrToYaml(hr)))),
						},
					},
				},
				Dependencies: utils.ConvertSliceToDNS1123(getTaskNamesFromHelmReleases(prevbucket)),
			}

			entry.DAG.Tasks = append(entry.DAG.Tasks, task)
		}
		prevbucket = bucket
	}

	if len(entry.DAG.Tasks) == 0 {
		return nil, fmt.Errorf("entry template must have at least one task")
	}

	return entry, nil
}

func (wc *ReverseWorkflowClient) validate() error {
	if wc.forwardWorkflow == nil {
		wc.Error(nil, "forward workflow object cannot be nil")
		return fmt.Errorf("forward workflow object cannot be nil")
	}
	return nil
}

func (wc *ReverseWorkflowClient) getReverseName() string {
	return fmt.Sprintf("%s-reverse", wc.forwardWorkflow.Name)
}
