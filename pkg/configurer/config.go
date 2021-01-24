package configurer

import (
	"errors"
	"fmt"
	"github.com/Azure/Orkestra/pkg/registry"
)

var (
	errEmptyKey         = errors.New("key cannot be an empty string")
	errEmptyRegistries  = errors.New("registries map cannot be empty")
	errRegistryNotFound = errors.New("registry entry not found in registries map")
)

type Controller struct {
	Registries map[string]*registry.Config `yaml:"registries"`
}

func (c *Controller) RegistryConfig(key string) (*registry.Config, error) {
	if key == "" {
		return nil, errEmptyKey
	}
	if c.Registries == nil || len(c.Registries) == 0 {
		return nil, errEmptyRegistries
	}

	v, ok := c.Registries[key]
	if !ok {
		return nil, fmt.Errorf("registry with key %s not found : %w", key, errRegistryNotFound)
	}

	return v, nil
}