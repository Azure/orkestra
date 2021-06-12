package meta

import (
	"errors"
)

var (
	InvalidSpecError              = errors.New("custom resource spec is invalid")
	WorkflowFailureError          = errors.New("workflow in failure status")
	HelmReleaseFailureStatusError = errors.New("helmrelease in failure status")

	ForwardWorkflowNotFound = errors.New("forward workflow not found")
)
