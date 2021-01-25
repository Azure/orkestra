package registry

import (
	"fmt"
	helmopv1 "github.com/fluxcd/helm-operator/pkg/apis/helm.fluxcd.io/v1"
	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"os"
	"path"
	"strings"
)

func Fetch(helmReleaseSpec helmopv1.HelmReleaseSpec, location string, logr logr.Logger) (string, error) {
	actionCfg := new(action.Configuration)

	settings := cli.New()
	helmDriver := "memory"
	if err := actionCfg.Init(settings.RESTClientGetter(), settings.Namespace(), helmDriver, logr.Info); err != nil {
		logr.Error(err, "unable to initialize action configuration")
		return "", fmt.Errorf("unable to initialize action configuration: %w", err)
	}

	var chartLocation string
	pullInstance := action.NewPull()
	pullInstance.Settings = settings
	if helmReleaseSpec.ChartPullSecret != nil {
		// todo(kushthedude): do we want to create a config
		logr.Info("found chartPullSecret")
	}
	var err error
	if location == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			logr.Error(err, "unable to get home directory")
			return "", fmt.Errorf("unable to get home directory: %w", err)
		}
		destFolder := "orkestra/charts"
		location = path.Join(homeDir, destFolder)
		errDir := os.MkdirAll(location, 0777)
		if errDir != nil {
			logr.Error(errDir, "unable to create directory")
			return "", fmt.Errorf("unable to create directory: %w", err)
		}
	} else {
		_, err := os.Stat(location)
		if os.IsNotExist(err) {
			errDir := os.MkdirAll(location, 0777)
			if errDir != nil {
				logr.Error(errDir, "unable to create directory")
				return "", fmt.Errorf("unable to create directory: %w", err)
			}
		}
	}

	pullInstance.Untar = true
	pullInstance.UntarDir = location
	chartLocation = location + "/" + helmReleaseSpec.ReleaseName

	var chartURL string
	if helmReleaseSpec.ChartSource.GitURL != "" {
		chartURL = helmReleaseSpec.ChartSource.GitURL + "/" + helmReleaseSpec.ChartSource.Path
	} else {
		chartURL = helmReleaseSpec.ChartSource.RepoURL + "/" + helmReleaseSpec.ChartSource.Name + "/" + helmReleaseSpec.ChartSource.Version
	}
	_, err = pullInstance.Run(chartURL)
	if err != nil {
		logr.Error(err, "failed to fetch chart")
		return "", fmt.Errorf("failed to fetch chart: %w", err)
	}

	return chartLocation, err
}

func Load(chartLocation string, cleanup bool, logr logr.Logger) (*chart.Chart, error) {
	actionConfig := new(action.Configuration)
	client := action.NewInstall(actionConfig)

	settings := cli.New()

	cp, err := client.ChartPathOptions.LocateChart(fmt.Sprint(chartLocation), settings)

	if err != nil {
		logr.Error(err, "failed to locate chart in specified directory")
		return nil, fmt.Errorf("failed to locate chart in specified directory: %w", err)
	}

	var chartRequested *chart.Chart

	if strings.Contains(chartLocation, ".tgz") {
		chartRequested, err = loader.LoadFile(cp)
	} else {
		chartRequested, err = loader.Load(cp)
	}

	if err != nil {
		logr.Error(err, "failed to load chart")
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	if !(chartRequested.Metadata.Type == "application" || chartRequested.Metadata.Type == "") {
		return nil, fmt.Errorf("%s charts are not installable", chartRequested.Metadata.Type)
	}

	if cleanup {
		defer os.RemoveAll(chartLocation)
	}

	return chartRequested, err
}
