package executor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type CustomForward struct {
	Image *corev1.Container
}

func (exec CustomForward) Reverse() Executor {
	return CustomReverse{exec.Image}
}

func (exec CustomForward) GetName() string {
	return "custom-forward-executor"
}

func (exec CustomForward) GetTemplate() v1alpha13.Template {
	return customBaseTemplate(exec.GetName(), Install, exec.Image)
}

func (exec CustomForward) GetTask(name string, dependencies []string, timeout, hrStr string, taskParams *apiextensionsv1.JSON) (v1alpha13.DAGTask, error) {
	return customBaseTask(exec.GetName(), name, dependencies, timeout, hrStr, taskParams)
}

type CustomReverse struct {
	Image *corev1.Container
}

func (exec CustomReverse) Reverse() Executor {
	return CustomForward{exec.Image}
}

func (exec CustomReverse) GetName() string {
	return "custom-reverse-executor"
}

func (exec CustomReverse) GetTemplate() v1alpha13.Template {
	return customBaseTemplate(exec.GetName(), Delete, exec.Image)
}

func (exec CustomReverse) GetTask(name string, dependencies []string, timeout, hrStr string, taskParams *apiextensionsv1.JSON) (v1alpha13.DAGTask, error) {
	return customBaseTask(exec.GetName(), name, dependencies, timeout, hrStr, taskParams)
}

func customBaseTemplate(executorName string, action Action, image *corev1.Container) v1alpha13.Template {
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
			Name:  image.Name,
			Image: image.Image,
			Args:  executorArgs,
		},
	}
}

func customBaseTask(executorName, name string, dependencies []string, timeout, hrStr string, taskParams *apiextensionsv1.JSON) (v1alpha13.DAGTask, error) {
	expectedParameters := &CustomParameters{}
	if taskParams == nil {
		return v1alpha13.DAGTask{}, fmt.Errorf("task parameters are required for the custom executor task")
	}
	if err := json.Unmarshal(taskParams.Raw, expectedParameters); err != nil {
		return v1alpha13.DAGTask{}, err
	}

	// Data must always be base64 encoded for the custom executor
	b64Data := base64.StdEncoding.EncodeToString(expectedParameters.Data.Raw)

	data := string(b64Data)

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
				{
					Name:  OpaqueDataArg,
					Value: utils.ToAnyStringPtr(data),
				},
			},
		},
		Dependencies: dependencies,
	}, nil
}

type CustomParameters struct {
	Data apiextensionsv1.JSON
}
