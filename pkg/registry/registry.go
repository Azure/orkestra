package registry

import (
	"fmt"

	"github.com/chartmuseum/helm-push/pkg/chartmuseum"
	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
)

const (
	defaultTargetDir = "/etc/orkestra/charts/pull/"
)

type helmConfig struct {
	pull *action.Pull
	push *chartmuseum.Client
}

type Client struct {
	l logr.InfoLogger
	// cfg stores the helm pull and push configurations
	cfg helmConfig
	// TargetDir is the location where the downloaded chart is saved
	TargetDir string
}

// Client is the constructor for the registry client
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{
		TargetDir: defaultTargetDir,
		cfg: helmConfig{
			pull: action.NewPull(),
			push: &chartmuseum.Client{},
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	err := c.init()
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

// TODO (nitishm) Implement

func (c *Client) PullChart(l logr.Logger, repoURL, name, version string) (*chart.Chart, error) {
	panic("Implement me")
}

// TODO (nitishm) Implement

func (c *Client) PushChart(l logr.Logger, repoURL string, ch *chart.Chart) error {
	panic("Implement me")
}
