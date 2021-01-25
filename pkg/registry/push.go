package registry

import (
	"errors"
	"fmt"
	cm "github.com/chartmuseum/helm-push/pkg/chartmuseum"
	"github.com/chartmuseum/helm-push/pkg/helm"
	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

func PushToStaging(ch *chart.Chart, logr logr.Logger, username, password, artifactoryURL string) error {
	if ch.Dependencies() == nil {
		logr.Info("No embedded subcharts found")
		return nil
	}
	for _, subChart := range ch.Dependencies() {
		resp, err := pushChart(subChart, username, password, artifactoryURL)
		if resp.StatusCode != 200 || err != nil {
			logr.Error(err, "unable to push chart to the upstream repository")
		}
	}
	return nil
}

func pushChart(ch *chart.Chart, username, password, repoURL string) (*http.Response, error) {
	var repo *helm.Repo
	var err error

	if regexp.MustCompile(`^https?://`).MatchString(repoURL) {
		repo, err = helm.TempRepoFromURL(repoURL)
		if err != nil {
			return nil, err
		}
		repoURL = repo.Config.URL
	}
	client, err := cm.NewClient(
		cm.URL(repo.Config.URL),
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
	tmp, err := ioutil.TempDir("", "helm-push-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	chartPackagePath, err := chartutil.Save(ch, tmp)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Pushing %s to %s...\n", filepath.Base(chartPackagePath), repoURL)
	resp, err := client.UploadChartPackage(chartPackagePath, true)
	if err != nil {
		return nil, err
	}

	return resp, nil
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
