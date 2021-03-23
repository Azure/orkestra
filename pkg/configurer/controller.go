package configurer

import (
	"github.com/Azure/Orkestra/pkg/registry"
)

type Controller struct {
	Registries              registry.RegistryMap `yaml:"registries"`
	DisableRemediation      bool                 `yaml:"disable-remediation"`
	CleanupDownloadedCharts bool                 `yaml:"cleanup-downloaded-charts"`
}

func (c *Controller) RegistryConfig(key string) (*registry.Config, error) {
	return c.Registries.RegistryConfig(key)
}
