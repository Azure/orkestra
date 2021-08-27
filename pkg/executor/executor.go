package executor

import (
	"fmt"
	"os"

	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// ExecutorAction defines the set of executor actions which can be performed on a helmrelease object
type Action string

const (
	Install Action = "install"
	Delete  Action = "delete"
)

const (
	DefaultTimeout   = "5m"
	ExecutorName     = "executor"
	ExecutorImage    = "azureorkestra/executor"
	ExecutorImageTag = "v0.4.1"

	HelmReleaseArg = "helmrelease"
	TimeoutArg     = "timeout"
)

func workflowServiceAccountName() string {
	if sa, ok := os.LookupEnv("WORKFLOW_SERVICEACCOUNT_NAME"); ok {
		return sa
	}
	return "orkestra"
}

func Default(templateName string, action Action) v1alpha13.Template {
	executorArgs := []string{"--spec", "{{inputs.parameters.helmrelease}}", "--action", string(action), "--timeout", "{{inputs.parameters.timeout}}", "--interval", "1s"}
	return v1alpha13.Template{
		Name:               templateName,
		ServiceAccountName: workflowServiceAccountName(),
		Inputs: v1alpha13.Inputs{
			Parameters: []v1alpha13.Parameter{
				{
					Name: HelmReleaseArg,
				},
				{
					Name:    TimeoutArg,
					Default: utils.ToAnyStringPtr(DefaultTimeout),
				},
			},
		},
		Executor: &v1alpha13.ExecutorConfig{
			ServiceAccountName: workflowServiceAccountName(),
		},
		Outputs: v1alpha13.Outputs{},
		Container: &corev1.Container{
			Name:  ExecutorName,
			Image: fmt.Sprintf("%s:%s", ExecutorImage, ExecutorImageTag),
			Args:  executorArgs,
		},
	}
}
