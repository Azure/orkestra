package executor

import (
	"fmt"
	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// Action defines the set of executor actions which can be performed on a helmrelease object
type Action string

const (
	HelmReleaseForwardExecutorName        = "helmrelease-forward-executor"
	HelmReleaseReverseExecutorName        = "helmrelease-reverse-executor"
	Install                        Action = "install"
	Delete                         Action = "delete"
)

type DefaultForward struct{}

func (exec DefaultForward) GetTemplate() v1alpha13.Template {
	return baseDefaultTemplate(HelmReleaseForwardExecutorName, Install)
}

func (exec DefaultForward) GetTask(name string, dependencies []string, timeout, hrStr *v1alpha13.AnyString) v1alpha13.DAGTask {
	return baseDefaultTask(HelmReleaseForwardExecutorName, name, dependencies, timeout, hrStr)
}

type DefaultReverse struct{}

func (exec DefaultReverse) GetTemplate() v1alpha13.Template {
	return baseDefaultTemplate(HelmReleaseReverseExecutorName, Delete)
}

func (exec DefaultReverse) GetTask(name string, dependencies []string, timeout, hrStr *v1alpha13.AnyString) v1alpha13.DAGTask {
	return baseDefaultTask(HelmReleaseReverseExecutorName, name, dependencies, timeout, hrStr)
}

func baseDefaultTemplate(executorName string, action Action) v1alpha13.Template {
	executorArgs := []string{"--spec", "{{inputs.parameters.helmrelease}}", "--action", string(action), "--timeout", "{{inputs.parameters.timeout}}", "--interval", "1s"}
	return v1alpha13.Template{
		Name:               executorName,
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

func baseDefaultTask(executorName, name string, dependencies []string, timeout, hrStr *v1alpha13.AnyString) v1alpha13.DAGTask {
	return v1alpha13.DAGTask{
		Name:     utils.ConvertToDNS1123(name),
		Template: executorName,
		Arguments: v1alpha13.Arguments{
			Parameters: []v1alpha13.Parameter{
				{
					Name:  HelmReleaseArg,
					Value: hrStr,
				},
				{
					Name:  TimeoutArg,
					Value: timeout,
				},
			},
		},
		Dependencies: dependencies,
	}
}
