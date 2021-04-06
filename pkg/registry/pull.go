package registry

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func (c *Client) PullChart(l logr.Logger, repoKey, repoPath, chartName, version string) (string, *chart.Chart, error) {
	// logic is derived from the "helm pull" command from the helm cli package
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
	c.cfg.pull.DestDir = c.TargetDir

	filePath := fmt.Sprintf("%s/%s-%s.tgz", c.TargetDir, chartName, version)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		l.V(3).Info("chart artifact not found in target directory - downloading")
		_, err = c.cfg.pull.Run(chartURL(rCfg.URL, repoPath, chartName, version))
		if err != nil {
			l.Error(err, "failed to pull chart from repo")
			return "", nil, fmt.Errorf("failed to pull chart from repoKey %s Name %s Version %s in registries map : %w", repoKey, chartName, version, err)
		}
	} else {
		l.V(3).Info("chart artifact found in target directory - skip downloading")
	}

	_, err = c.cfg.pull.ChartPathOptions.LocateChart(filePath, c.cfg.pull.Settings)
	if err != nil {
		l.Error(err, "failed to locate chart in filesystem")
		return "", nil, fmt.Errorf("failed to locate chart in filesystem at path %s : %w", filePath, err)
	}

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
