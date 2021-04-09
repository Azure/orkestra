package registry

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chartmuseum/helm-push/pkg/chartmuseum"
	"github.com/chartmuseum/helm-push/pkg/helm"
	"github.com/go-logr/logr"
	"github.com/gofrs/flock"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
)

const (
	defaultTargetDir = "/Users/jonathaninnis/orkestra"
)

var (
	errEmptyKey         = errors.New("key cannot be an empty string")
	errEmptyRegistries  = errors.New("registries map cannot be nil or empty")
	errRegistryNotFound = errors.New("registry entry not found in registries map")
)

type helmActionConfig struct {
	pull *action.Pull
	push *chartmuseum.Client
}

type Client struct {
	l logr.InfoLogger
	// rfile is the handle to the helm repo file configuration
	rfile *repo.File
	// repoFilePath is the location of the helm repo file
	repoFilePath string
	// cfg stores the helm pull and push configurations
	cfg helmActionConfig
	// TargetDir is the location where the downloaded chart is saved
	TargetDir string
	// settings
	settings *cli.EnvSettings

	// Registries maps the registry name to it's configuration data
	registries map[string]*Config
}

// NewClient is the constructor for the registry client
func NewClient(l logr.InfoLogger, opts ...Option) (*Client, error) {
	cm, err := chartmuseum.NewClient()
	if err != nil {
		return nil, err
	}

	c := &Client{
		l:         l,
		TargetDir: defaultTargetDir,
		rfile:     repo.NewFile(),
		cfg: helmActionConfig{
			pull: action.NewPull(),
			push: cm,
		},
		settings:   cli.New(),
		registries: make(map[string]*Config),
	}

	for _, opt := range opts {
		opt(c)
	}

	err = c.init()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Client) init() error {
	c.repoFilePath = c.settings.RepositoryConfig

	// Initialize the helm repo file
	repoFile := c.settings.RepositoryConfig
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

	if err := yaml.Unmarshal(b, c.rfile); err != nil {
		return err
	}

	// If no TargetDir option was passed, set to default location
	if c.TargetDir == "" {
		c.TargetDir = defaultTargetDir
	}
	// Set destination directory where we download the chart
	c.cfg.pull.DestDir = c.TargetDir

	// Initialize the pull and push clients
	// Pull client config
	actionCfg := new(action.Configuration)
	helmDriver := "memory"

	if err := actionCfg.Init(c.settings.RESTClientGetter(), c.settings.Namespace(), helmDriver, c.l.Info); err != nil {
		return fmt.Errorf("unable to initialize action configuration: %w", err)
	}

	c.cfg.pull.Settings = c.settings

	// Push Client
	// no init required

	return nil
}

func (c *Client) AddRepo(cfg *Config) error {
	e := repo.Entry{
		Name:     cfg.Name,
		URL:      cfg.URL,
		Username: cfg.Username,
		Password: cfg.Password,
		CertFile: cfg.CertFile,
		KeyFile:  cfg.KeyFile,
		CAFile:   cfg.CaFile,
	}

	r, err := repo.NewChartRepository(&e, getter.All(c.settings))
	if err != nil {
		return err
	}

	if _, err := r.DownloadIndexFile(); err != nil {
		return fmt.Errorf("looks like %q is not a valid chart repository or cannot be reached : %w", cfg.URL, err)
	}

	c.rfile.Update(&e)

	if err := c.rfile.WriteFile(c.repoFilePath, 0644); err != nil {
		return err
	}

	c.registries[cfg.Name] = cfg

	return nil
}

func (c *Client) RegistryConfig(name string) (*Config, error) {
	if name == "" {
		return nil, errEmptyKey
	}
	if c.registries == nil {
		return nil, errEmptyRegistries
	}
	v, ok := c.registries[name]
	if !ok {
		return nil, errRegistryNotFound
	}

	return v, nil
}

func chartURL(repo, repoPath, chart, version string) string {
	if repoPath != "" {
		chart = strings.Trim(repoPath, "/") + "/" + chart
	}

	s := fmt.Sprintf("%s/%s-%s.tgz",
		strings.Trim(repo, "/"),
		strings.Trim(chart, "/"),
		version,
	)

	// Validate the URL
	if u, err := url.ParseRequestURI(s); u != nil || err != nil {
		return s
	}
	return ""
}

func SaveChartPackage(ch *chart.Chart, dir string) (string, error) {
	return helm.CreateChartPackage(&helm.Chart{V3: ch}, dir)
}

func GetHelmRepoConfig(app *orkestrav1alpha1.Application, c client.Client) (*Config, error) {
	cfg := &Config{
		Name: app.Name,
		URL:  app.Spec.Chart.RepoURL,
	}

	if app.Spec.Chart.HelmRepoSecretRef != nil {
		if app.Spec.Chart.HelmRepoSecretRef.Namespace == "" {
			app.Spec.Chart.HelmRepoSecretRef.Namespace = "default"
		}
		creds := &v1.Secret{}
		key := types.NamespacedName{
			Name:      app.Spec.Chart.HelmRepoSecretRef.Name,
			Namespace: app.Spec.Chart.HelmRepoSecretRef.Namespace,
		}

		err := c.Get(context.Background(), key, creds)
		if err != nil {
			return nil, err
		}

		data := creds.Data

		if v, ok := data["username"]; ok {
			cfg.Username = string(v)
		}

		if v, ok := data["password"]; ok {
			cfg.Password = string(v)
		}

		if v, ok := data["username"]; ok {
			cfg.Username = string(v)
		}

		if v, ok := data["tls.crt"]; ok {
			cfg.CertFile = string(v)
		}

		if v, ok := data["tls.key"]; ok {
			cfg.KeyFile = string(v)
		}

		if v, ok := data["ca.crt"]; ok {
			cfg.CaFile = string(v)
		}
	}
	return cfg, nil
}

type CredentialsObjectReference struct {
	// Name of the referent
	Name string `yaml:"name" json:"name,omitempty"`

	// Namespace of the referent,
	Namespace string `yaml:"namespace" json:"namespace,omitempty"`
}
