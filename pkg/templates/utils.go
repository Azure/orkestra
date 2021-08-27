package templates

import (
	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultTimeout = "5m"
)

func getTimeout(t *v1.Duration) *v1alpha13.AnyString {
	if t == nil {
		return utils.ToAnyStringPtr(DefaultTimeout)
	}
	tm := utils.ToAnyString(t.Duration.String())
	return &tm
}
