package registry

import (
	"errors"
	"github.com/chartmuseum/helm-push/pkg/helm"
	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/chart"
	"github.com/chartmuseum/helm-push/cmd/helmpush"
	cm "github.com/chartmuseum/helm-push/pkg/chartmuseum"
	"io/ioutil"
	"net/http"
)

func PushToStaging(ch *chart.Chart, artifactoryURL string, logr logr.Logger) error {
	if ch.Dependencies() == nil {
		logr.Info("No embedded subcharts found")
		return nil
	}
	for _, subChart := range ch.Dependencies() {
		resp, err := pushChart(subChart, artifactoryURL)
	}
	return nil
}

func pushChart(ch *chart.Chart, url string) (*http.Response, error) {
	client, err := cm.NewClient(
		cm.URL(url),
		cm.Username(username),
		cm.Password(password),
		)
	if err != nil {
		return nil, err
	}

	index, err := helm.GetIndexByRepo(repo, getIndexDownloader(client))
	if err != nil {
		return nil, err
	}
	client.Option(cm.ContextPath(index.ServerInfo.ContextPath))

	return nil, nil
}

func getIndexDownloader(client *cm.Client) helm.IndexDownloader {
	return func() ([]byte, error) {
		resp, err := client.DownloadFile("index.yaml")
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			return nil, errors.New("couldn't fetch index.yaml")
		}
		return b, nil
	}
}
