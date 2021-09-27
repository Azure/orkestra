package templates

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultTimeout = "5m"
)

func getTimeout(t *v1.Duration) string {
	if t == nil {
		return DefaultTimeout
	}
	return t.Duration.String()
}
