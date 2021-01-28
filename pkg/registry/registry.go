package registry

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/chartmuseum/helm-push/pkg/chartmuseum"
	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
)

const (
	defaultTargetDir = "/etc/orkestra/charts/pull/"
)

var (
	errEmptyKey         = errors.New("key cannot be an empty string")
	errEmptyRegistries  = errors.New("registries map cannot be nil or empty")
	errRegistryNotFound = errors.New("registry entry not found in registries map")
)

type RegistryMap map[string]*Config

func (rm RegistryMap) RegistryConfig(key string) (*Config, error) {
	if key == "" {
		return nil, errEmptyKey
	}
	if rm == nil || len(rm) == 0 {
		return nil, errEmptyRegistries
	}

	v, ok := rm[key]
	if !ok {
		return nil, fmt.Errorf("registry with key %s not found : %w", key, errRegistryNotFound)
	}

	return v, nil
}

type helmActionConfig struct {
	pull *action.Pull
	push *chartmuseum.Client
}

type Client struct {
	l logr.InfoLogger
	// cfg stores the helm pull and push configurations
	cfg helmActionConfig
	// TargetDir is the location where the downloaded chart is saved
	TargetDir string

	registries RegistryMap
}

// NewClient is the constructor for the registry client
func NewClient(l logr.InfoLogger, registries map[string]*Config, opts ...Option) (*Client, error) {
	cm, err := chartmuseum.NewClient()
	if err != nil {
		return nil, err
	}

	c := &Client{
		l:         l,
		TargetDir: defaultTargetDir,
		cfg: helmActionConfig{
			pull: action.NewPull(),
			push: cm,
		},
		registries: registries,
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
	// If no TargetDir option was passed, set to default location
	if c.TargetDir == "" {
		c.TargetDir = defaultTargetDir
	}
	// Set destination directory where we download the chart
	c.cfg.pull.DestDir = c.TargetDir

	// Initialize the pull and push clients
	// Pull client config
	actionCfg := new(action.Configuration)
	settings := cli.New()
	helmDriver := "memory"

	if err := actionCfg.Init(settings.RESTClientGetter(), settings.Namespace(), helmDriver, c.l.Info); err != nil {
		return fmt.Errorf("unable to initialize action configuration: %w", err)
	}

	c.cfg.pull.Settings = settings

	// Push Client
	// no init required

	return nil
}

func (c *Client) PullChart(l logr.Logger, repoKey, chartName, version string) (string, *chart.Chart, error) {
	l.WithValues("repo-key", repoKey, "chart-name", chartName, "chart-version", version)

	l.V(3).Info("pulling chart")

	rCfg, err := c.registries.RegistryConfig(repoKey)
	if err != nil {
		l.Error(err, "failed to find registry with provided key in registries map")
		return "", nil, fmt.Errorf("failed to find registry with repoKey %s Name %s Version %s in registries map : %w", repoKey, chartName, version, err)
	}

	c.cfg.pull.Username = rCfg.Username
	c.cfg.pull.Password = rCfg.Password
	c.cfg.pull.CaFile = rCfg.CaFile
	c.cfg.pull.CaFile = rCfg.CaFile
	c.cfg.pull.CertFile = rCfg.CertFile
	c.cfg.pull.KeyFile = rCfg.KeyFile

	filePath := fmt.Sprintf("%s/%s-%s.tgz", c.TargetDir, chartName, version)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		l.V(3).Info("chart artifact not found in target directory - downloading")
		_, err = c.cfg.pull.Run(chartURL(rCfg.URL, chartName, version))
		if err != nil {
			l.Error(err, "failed to pull chart from repo")
			return "", nil, fmt.Errorf("failed to pull chart from repoKey %s Name %s Version %s in registries map : %w", repoKey, chartName, version, err)
		}
	} else {
		l.V(3).Info("chart artifact found in target directory - skip downloading")
	}

	c.cfg.pull.ChartPathOptions.LocateChart(filePath, c.cfg.pull.Settings)

	var ch *chart.Chart

	ch, err = loader.LoadFile(filePath)

	if err != nil {
		l.Error(err, "failed to load chart")
		return "", nil, fmt.Errorf("failed to load chart: %w", err)
	}

	if !(ch.Metadata.Type == "application" || ch.Metadata.Type == "") {
		return "", nil, fmt.Errorf("%s charts are not installable", ch.Metadata.Type)
	}

	return filePath, ch, nil
}

// TODO (nitishm) Implement me
func (c *Client) PushChart(l logr.Logger, repoName string, ch *chart.Chart) error {
	l.Info("pushing chart")
	return nil
}

func chartURL(repo, chart, version string) string {
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
