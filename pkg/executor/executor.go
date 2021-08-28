package executor

import (
	"os"

	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
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

type Executor interface {
	GetTemplate() v1alpha13.Template
	GetTask(name string, dependencies []string, timeout, hrStr *v1alpha13.AnyString) v1alpha13.DAGTask
}
