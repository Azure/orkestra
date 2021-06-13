package meta

import (
	"errors"
)

var (
	ErrInvalidSpec              = errors.New("custom resource spec is invalid")
	ErrWorkflowFailure          = errors.New("workflow in failure status")
	ErrHelmReleaseStatusFailure = errors.New("helmrelease in failure status")

	ErrForwardWorkflowNotFound = errors.New("forward workflow not found")
	ErrPreviousSpecNotSet      = errors.New("failed to generate rollback workflow, previous spec is unset")
)
