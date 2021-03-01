package workflow

import (
	"context"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/go-logr/logr"
)

const (
	Project        = "orkestra"
	OwnershipLabel = "owner"
	HeritageLabel  = "heritage"
)

type Engine interface {
	// Generate the object required by the workflow engine
	Generate(ctx context.Context, l logr.Logger, ns string, g *v1alpha1.ApplicationGroup) error
	// Submit the object required by the workflow engine generated by the Generate method
	Submit(ctx context.Context, l logr.Logger, g *v1alpha1.ApplicationGroup) error
}
