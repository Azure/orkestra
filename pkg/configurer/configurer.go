package configurer

import (
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/spf13/viper"
)

const (
	defaultConfigPath = "/etc/controller/config.yaml"
)

type Configurer struct {
	v    *viper.Viper
	Ctrl *Controller
}

func NewConfigurer(cfgPath string) (*Configurer, error) {
	v := viper.New()
	if cfgPath == "" {
		cfgPath = defaultConfigPath
	}

	v.SetConfigFile(cfgPath)
	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}

	ctrlCfg := &Controller{
		Registries: make(map[string]*registry.Config),
	}

	err = v.Unmarshal(ctrlCfg)
	if err != nil {
		return nil, err
	}

	return &Configurer{
		v:    v,
		Ctrl: ctrlCfg,
	}, nil
}
