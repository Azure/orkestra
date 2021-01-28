package configurer

import (
	"github.com/Azure/Orkestra/pkg/registry"
)

type Controller struct {
	Registries registry.RegistryMap `yaml:"registries"`
	// Cleanup the downloaded charts after they are pushed to staging repo
	Cleanup bool
}

func (c *Controller) RegistryConfig(key string) (*registry.Config, error) {
	return c.Registries.RegistryConfig(key)
}
