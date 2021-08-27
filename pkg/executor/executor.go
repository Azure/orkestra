package executor

import (
	"fmt"
	"github.com/Azure/Orkestra/pkg/workflow"
	"os"

	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// ExecutorAction defines the set of executor actions which can be performed on a helmrelease object
type ExecutorAction string

const (
	Install ExecutorAction = "install"
	Delete  ExecutorAction = "delete"
	DefaultTimeout string = "5m"
)

func workflowServiceAccountName() string {
	if sa, ok := os.LookupEnv("WORKFLOW_SERVICEACCOUNT_NAME"); ok {
		return sa
	}
	return "orkestra"
}

func Default(templateName string, action ExecutorAction) v1alpha13.Template {
	executorArgs := []string{"--spec", "{{inputs.parameters.helmrelease}}", "--action", string(action), "--timeout", "{{inputs.parameters.timeout}}", "--interval", "1s"}
	return v1alpha13.Template{
		Name:               templateName,
		ServiceAccountName: workflowServiceAccountName(),
		Inputs: v1alpha13.Inputs{
			Parameters: []v1alpha13.Parameter{
				{
					Name: workflow.HelmReleaseArg,
				},
				{
					Name:    workflow.TimeoutArg,
					Default: utils.ToAnyStringPtr(DefaultTimeout),
				},
			},
		},
		Executor: &v1alpha13.ExecutorConfig{
			ServiceAccountName: workflowServiceAccountName(),
		},
		Outputs: v1alpha13.Outputs{},
		Container: &corev1.Container{
			Name:  workflow.ExecutorName,
			Image: fmt.Sprintf("%s:%s", workflow.ExecutorImage, workflow.ExecutorImageTag),
			Args:  executorArgs,
		},
	}
}
