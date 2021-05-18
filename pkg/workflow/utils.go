package workflow

import (
	"github.com/Azure/Orkestra/pkg/utils"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

func convertSliceToDNS1123(in []string) []string {
	out := []string{}
	for _, s := range in {
		out = append(out, utils.ConvertToDNS1123(s))
	}
	return out
}

func strToStrPtr(in string) *string {
	return &in
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

func getTimeout(t *v1.Duration) *string {
	if t == nil {
		return strToStrPtr(DefaultTimeout)
	}
	tm := t.Duration.String()
	return &tm
}

func hrToYAML(hr fluxhelmv2beta1.HelmRelease) string {
	b, err := yaml.Marshal(hr)
	if err != nil {
		return ""
	}

	return string(b)
}
