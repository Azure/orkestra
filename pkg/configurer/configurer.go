package configurer

import (
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/spf13/viper"
)
const (
	defaultConfig = "/etc/controller/config.yaml"
)

type Configurer struct {
	cfg  *viper.Viper
	Ctrl *Controller
}

func NewConfigurer(loc string) (*Configurer, error) {
	v := viper.New()
	if loc == "" {
		loc = defaultConfig
	}

	v.SetConfigFile(loc)
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
		cfg:  v,
		Ctrl: ctrlCfg,
	}, nil
}