package workflow

import (
	"errors"
)

var (
	ErrNoNodesFound      = errors.New("no nodes found in the graph")
	ErrEntryNodeNotFound = errors.New("\"entry\" node not found")
)
