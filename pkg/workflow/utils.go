package workflow

import (
	"github.com/Azure/Orkestra/pkg/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

func getTimeout(t *v1.Duration) *string {
	if t == nil {
		return utils.ToStrPtr(DefaultTimeout)
	}
	tm := t.Duration.String()
	return &tm
}

func workflowNamespace() string {
	if ns, ok := os.LookupEnv("WORKFLOW_NAMESPACE"); ok {
		return ns
	}
	return "orkestra"
}

func workflowServiceAccountName() string {
	if sa, ok := os.LookupEnv("WORKFLOW_SERVICEACCOUNT_NAME"); ok {
		return sa
	}
	return "orkestra"
}
