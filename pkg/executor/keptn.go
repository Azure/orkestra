package executor

import (
	"encoding/json"
	"fmt"
	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	KeptnImage = "azureorkestra/keptn-executor"
	KeptnTag   = "v0.1.0"
)

const (
	configMapName      = "configMapName"
	configMapNamespace = "configMapNamespace"
)

type KeptnForward struct{}

func (exec KeptnForward) Reverse() Executor {
	return KeptnReverse{}
}

func (exec KeptnForward) GetName() string {
	return "keptn-forward-executor"
}

func (exec KeptnForward) GetTemplate() v1alpha13.Template {
	return keptnBaseTemplate(exec.GetName(), Install)
}

func (exec KeptnForward) GetTask(name string, dependencies []string, timeout, hrStr string, taskParams *apiextensionsv1.JSON) (v1alpha13.DAGTask, error) {
	return keptnBaseTask(exec.GetName(), name, dependencies, timeout, hrStr, taskParams)
}

type KeptnReverse struct{}

func (exec KeptnReverse) Reverse() Executor {
	return KeptnForward{}
}

func (exec KeptnReverse) GetName() string {
	return "keptn-reverse-executor"
}

func (exec KeptnReverse) GetTemplate() v1alpha13.Template {
	return keptnBaseTemplate(exec.GetName(), Delete)
}

func (exec KeptnReverse) GetTask(name string, dependencies []string, timeout, hrStr string, taskParams *apiextensionsv1.JSON) (v1alpha13.DAGTask, error) {
	return keptnBaseTask(exec.GetName(), name, dependencies, timeout, hrStr, taskParams)
}

func keptnBaseTemplate(executorName string, action Action) v1alpha13.Template {
	executorArgs := []string{"--spec", "{{inputs.parameters.helmrelease}}", "--action", string(action), "--configmap-name", "{{inputs.parameters.configMapName}}", "configmap-namespace", "{{inputs.parameters.configMapNamespace}}", "--timeout", "{{inputs.parameters.timeout}}", "--interval", "1s"}
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
					Name: configMapName,
				},
				{
					Name: configMapNamespace,
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

func keptnBaseTask(executorName, name string, dependencies []string, timeout, hrStr string, taskParams *apiextensionsv1.JSON) (v1alpha13.DAGTask, error) {
	expectedParameters := &KeptnParameters{}
	if taskParams == nil {
		return v1alpha13.DAGTask{}, fmt.Errorf("task parameters are required for the keptn executor task")
	}
	if err := json.Unmarshal(taskParams.Raw, expectedParameters); err != nil {
		return v1alpha13.DAGTask{}, err
	}
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
					Name:  configMapName,
					Value: utils.ToAnyStringPtr(expectedParameters.ConfigMapRef.Name),
				},
				{
					Name:  configMapNamespace,
					Value: utils.ToAnyStringPtr(expectedParameters.ConfigMapRef.Namespace),
				},
			},
		},
		Dependencies: dependencies,
	}, nil
}

type KeptnParameters struct {
	ConfigMapRef corev1.ObjectReference `json:"configMapRef"`
}
