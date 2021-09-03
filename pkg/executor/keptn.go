package executor

import (
	"fmt"
	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

const (
	KeptnImage = "azureorkestra/keptn-executor"
	KeptnTag   = "v0.1.0"
)

type KeptnForward struct{}

func (exec KeptnForward) Reverse() Executor {
	return HelmReleaseReverse{}
}

func (exec KeptnForward) GetName() string {
	return "keptn-forward-executor"
}

func (exec KeptnForward) GetTemplate() v1alpha13.Template {
	return keptnBaseTemplate(exec.GetName(), Install)
}

func (exec KeptnForward) GetTask(name string, dependencies []string, timeout, hrStr *v1alpha13.AnyString) v1alpha13.DAGTask {
	return keptnBaseTask(exec.GetName(), name, dependencies, timeout, hrStr)
}

type KeptnReverse struct{}

func (exec KeptnReverse) Reverse() Executor {
	return HelmReleaseForward{}
}

func (exec KeptnReverse) GetName() string {
	return "keptn-reverse-executor"
}

func (exec KeptnReverse) GetTemplate() v1alpha13.Template {
	return keptnBaseTemplate(exec.GetName(), Delete)
}

func (exec KeptnReverse) GetTask(name string, dependencies []string, timeout, hrStr *v1alpha13.AnyString) v1alpha13.DAGTask {
	return keptnBaseTask(exec.GetName(), name, dependencies, timeout, hrStr)
}

func keptnBaseTemplate(executorName string, action Action) v1alpha13.Template {
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
			Name:  executorName,
			Image: fmt.Sprintf("%s:%s", KeptnImage, KeptnTag),
			Args:  executorArgs,
		},
	}
}

func keptnBaseTask(executorName, name string, dependencies []string, timeout, hrStr *v1alpha13.AnyString) v1alpha13.DAGTask {
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
