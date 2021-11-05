package executor

import (
	"fmt"

	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	data = "data"
)

type CustomForward struct {
	ImageName string `json:"imageName,omitempty"`
	ImageTag  string `json:"imageTag,omitempty"`
}

func (exec CustomForward) Reverse() Executor {
	return CustomReverse{}
}

func (exec CustomForward) GetName() string {
	return "helmrelease-forward-executor"
}

func (exec CustomForward) GetTemplate() v1alpha13.Template {
	return customBaseTemplate(exec.GetName(), Install, exec.ImageName, exec.ImageTag)
}

func (exec CustomForward) GetTask(name string, dependencies []string, timeout, hrStr string, taskParams *apiextensionsv1.JSON) (v1alpha13.DAGTask, error) {
	return customBaseTask(exec.GetName(), name, dependencies, timeout, hrStr), nil
}

type CustomReverse struct {
	ImageName string
	ImageTag  string
}

func (exec CustomReverse) Reverse() Executor {
	return CustomForward{}
}

func (exec CustomReverse) GetName() string {
	return "helmrelease-reverse-executor"
}

func (exec CustomReverse) GetTemplate() v1alpha13.Template {
	return customBaseTemplate(exec.GetName(), Delete, exec.ImageName, exec.ImageTag)
}

func (exec CustomReverse) GetTask(name string, dependencies []string, timeout, hrStr string, taskParams *apiextensionsv1.JSON) (v1alpha13.DAGTask, error) {
	return customBaseTask(exec.GetName(), name, dependencies, timeout, hrStr), nil
}

func customBaseTemplate(executorName string, action Action, imageName string, imageTag string) v1alpha13.Template {
	executorArgs := []string{"--spec", "{{inputs.parameters.helmrelease}}", "--action", string(action), "--data", "{{inputs.parameters.data}}", "--timeout", "{{inputs.parameters.timeout}}", "--interval", "1s"}
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
				{
					Name: OpaqueDataArg,
				},
			},
		},
		Executor: &v1alpha13.ExecutorConfig{
			ServiceAccountName: workflowServiceAccountName(),
		},
		Container: &corev1.Container{
			Name:  ExecutorName,
			Image: fmt.Sprintf("%s:%s", imageName, imageTag),
			Args:  executorArgs,
		},
	}
}

func customBaseTask(executorName, name string, dependencies []string, timeout, hrStr string) v1alpha13.DAGTask {
	return v1alpha13.DAGTask{
		Name:     utils.ConvertToDNS1123(name),
		Template: executorName,
		Arguments: v1alpha13.Arguments{
			Parameters: []v1alpha13.Parameter{
				{
					Name:  HelmReleaseArg,
					Value: utils.ToAnyStringPtr(hrStr),
				},
				{
					Name:  TimeoutArg,
					Value: utils.ToAnyStringPtr(timeout),
				},
			},
		},
		Dependencies: dependencies,
	}
}
