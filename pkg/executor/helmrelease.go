package executor

import (
	"fmt"
	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

const (
	HelmReleaseImage = "azureorkestra/executor"
	HelmReleaseTag   = "v0.4.2"
)

type HelmReleaseForward struct{}

func (exec HelmReleaseForward) Reverse() Executor {
	return HelmReleaseReverse{}
}

func (exec HelmReleaseForward) GetName() string {
	return "helmrelease-forward-executor"
}

func (exec HelmReleaseForward) GetTemplate() v1alpha13.Template {
	return helmReleaseBaseTemplate(exec.GetName(), Install)
}

func (exec HelmReleaseForward) GetTask(name string, dependencies []string, timeout, hrStr *v1alpha13.AnyString) v1alpha13.DAGTask {
	return helmReleaseBaseTask(exec.GetName(), name, dependencies, timeout, hrStr)
}

type HelmReleaseReverse struct{}

func (exec HelmReleaseReverse) Reverse() Executor {
	return HelmReleaseForward{}
}

func (exec HelmReleaseReverse) GetName() string {
	return "helmrelease-reverse-executor"
}

func (exec HelmReleaseReverse) GetTemplate() v1alpha13.Template {
	return helmReleaseBaseTemplate(exec.GetName(), Delete)
}

func (exec HelmReleaseReverse) GetTask(name string, dependencies []string, timeout, hrStr *v1alpha13.AnyString) v1alpha13.DAGTask {
	return helmReleaseBaseTask(exec.GetName(), name, dependencies, timeout, hrStr)
}

func helmReleaseBaseTemplate(executorName string, action Action) v1alpha13.Template {
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
		Container: &corev1.Container{
			Name:  ExecutorName,
			Image: fmt.Sprintf("%s:%s", HelmReleaseImage, HelmReleaseTag),
			Args:  executorArgs,
		},
	}
}

func helmReleaseBaseTask(executorName, name string, dependencies []string, timeout, hrStr *v1alpha13.AnyString) v1alpha13.DAGTask {
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
