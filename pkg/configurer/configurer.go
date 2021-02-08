package configurer

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/gofrs/flock"
	"github.com/spf13/viper"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
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
		Cleanup:    false,
	}

	err = v.Unmarshal(ctrlCfg)
	if err != nil {
		return nil, err
	}

	err = setupRegistryRepos(ctrlCfg.Registries)
	if err != nil {
		return nil, err
	}

	return &Configurer{
		v:    v,
		Ctrl: ctrlCfg,
	}, nil
}

func setupRegistryRepos(registries map[string]*registry.Config) error {
	settings := cli.New()
	repoFile := settings.RepositoryConfig
	err := os.MkdirAll(filepath.Dir(repoFile), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	// Acquire a file lock for process synchronization
	fileLock := flock.New(strings.Replace(repoFile, filepath.Ext(repoFile), ".lock", 1))
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		defer fileLock.Unlock() //nolint:errcheck
	}
	if err != nil {
		return err
	}

	b, err := ioutil.ReadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return err
	}

	for name, cfg := range registries {
		c := repo.Entry{
			Name:     name,
			URL:      cfg.URL,
			Username: cfg.Username,
			Password: cfg.Password,
			CertFile: cfg.CertFile,
			KeyFile:  cfg.KeyFile,
			CAFile:   cfg.CaFile,
		}

		r, err := repo.NewChartRepository(&c, getter.All(settings))
		if err != nil {
			return err
		}

		if _, err := r.DownloadIndexFile(); err != nil {
			return fmt.Errorf("looks like %q is not a valid chart repository or cannot be reached : %w", c.URL, err)
		}

		f.Update(&c)

		if err := f.WriteFile(repoFile, 0644); err != nil {
			return err
		}
	}

	return nil
}
