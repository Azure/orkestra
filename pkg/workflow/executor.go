package workflow

import (
	"fmt"

	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// ExecutorAction defines the set of executor actions which can be performed on a helmrelease object
type ExecutorAction string

const (
	Install ExecutorAction = "install"
	Delete  ExecutorAction = "delete"
)

func defaultExecutor() v1alpha13.Template {
	executorArgs := []string{"--spec", "{{inputs.parameters.helmrelease}}", "--action", "{{inputs.parameters.action}}", "--timeout", "{{inputs.parameters.timeout}}", "--interval", "1s"}
	return v1alpha13.Template{
		Name:               "default",
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
				{
					Name: ActionArg,
				},
			},
		},
		Executor: &v1alpha13.ExecutorConfig{
			ServiceAccountName: workflowServiceAccountName(),
		},
		Outputs: v1alpha13.Outputs{},
		Container: &corev1.Container{
			Name:            ExecutorName,
			ImagePullPolicy: corev1.PullAlways,
			Image:           fmt.Sprintf("%s:%s", ExecutorImage, ExecutorImageTag),
			Args:            executorArgs,
		},
	}
}

func keptnExecutor() v1alpha13.Template {
	executorArgs := []string{"--spec", "{{inputs.parameters.helmrelease}}", "--action", "{{inputs.parameters.action}}", "--timeout", "{{inputs.parameters.timeout}}", "--interval", "1s", "--configmap-name", "foobar", "--configmap-namespace", "default"}
	return v1alpha13.Template{
		Name:               KeptnExecutorName,
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
				{
					Name: ActionArg,
				},
			},
		},
		Executor: &v1alpha13.ExecutorConfig{
			ServiceAccountName: workflowServiceAccountName(),
		},
		Outputs: v1alpha13.Outputs{},
		Container: &corev1.Container{
			Name:            ExecutorName,
			ImagePullPolicy: corev1.PullAlways,
			Image:           fmt.Sprintf("%s:%s", KeptnExecutor, KeptnExecutorImageTag),
			Args:            executorArgs,
		},
	}
}

func chainedDefaultKeptnExecutor(templateName string, action ExecutorAction) v1alpha13.Template {
	return v1alpha13.Template{
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
		Name: templateName,
		DAG: &v1alpha13.DAGTemplate{
			Tasks: []v1alpha13.DAGTask{
				{
					Name:     defaultExecutor().Name,
					Template: defaultExecutor().Name,
					Arguments: v1alpha13.Arguments{
						Parameters: []v1alpha13.Parameter{
							{
								Name:  HelmReleaseArg,
								Value: utils.ToAnyStringPtr("{{inputs.parameters.helmrelease}}"),
							},
							{
								Name:  TimeoutArg,
								Value: utils.ToAnyStringPtr("{{inputs.parameters.timeout}}"),
							},
							{
								Name:  ActionArg,
								Value: utils.ToAnyStringPtr(string(action)),
							},
						},
					},
				},
				{
					Name:     keptnExecutor().Name,
					Template: keptnExecutor().Name,
					Arguments: v1alpha13.Arguments{
						Parameters: []v1alpha13.Parameter{
							{
								Name:  HelmReleaseArg,
								Value: utils.ToAnyStringPtr("{{inputs.parameters.helmrelease}}"),
							},
							{
								Name:  TimeoutArg,
								Value: utils.ToAnyStringPtr("{{inputs.parameters.timeout}}"),
							},
							{
								Name:  ActionArg,
								Value: utils.ToAnyStringPtr(string(action)),
							},
						},
					},
					Dependencies: []string{defaultExecutor().Name},
				},
			},
		},
	}
}
