package workflow

import (
	"errors"
)

var (
	ErrNoNodesFound       = errors.New("no nodes found in the graph")
	ErrEntryNodeNotFound  = errors.New("\"entry\" node not found")
	ErrInvalidInputsPtr   = errors.New("invalid Node.Status.Inputs pointer")
	ErrNilParametersSlice = errors.New("nil/empty Node.Status.Inputs.Parameters slice")
	ErrInvalidValuePtr    = errors.New("invalid Node.Status.Inputs.Parameters[0].Value pointer")
)
