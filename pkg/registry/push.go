package registry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/chartmuseum/helm-push/pkg/chartmuseum"
	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/chart"
)

// PushChart pushes the chart to the repository specified by the repoKey. The repository setting is fetched from the associated registry config file
func (c *Client) PushChart(l logr.Logger, repoKey, pkgPath string, ch *chart.Chart) error {
	// logic is derived from the "helm push" extension from the chartmuseum folks
	chartName := ch.Name()
	version := ch.Metadata.Version

	l.WithValues("repo-key", repoKey, "chart-name", chartName, "chart-version", version)
	l.V(3).Info("pushing chart")

	rCfg, err := c.RegistryConfig(repoKey)
	if err != nil {
		l.Error(err, "failed to find registry with provided key in registries map")
		return fmt.Errorf("failed to find registry with repoKey %s Name %s Version %s in registries map : %w", repoKey, chartName, version, err)
	}

	// Set the URL to the port-forward address:port of chartmuseum (http://localhost:8080)
	if url := os.Getenv("CI_ENVTEST_CHARTMUSEUM_URL"); url != "" {
		rCfg.URL = url
	}

	c.cfg.push, err = chartmuseum.NewClient(
		chartmuseum.URL(rCfg.URL),
		chartmuseum.Username(rCfg.Username),
		chartmuseum.Password(rCfg.Password),
		chartmuseum.AccessToken(rCfg.AccessToken),
		chartmuseum.AuthHeader(rCfg.AuthHeader),
		chartmuseum.CAFile(rCfg.CaFile),
		chartmuseum.CertFile(rCfg.CertFile),
		chartmuseum.KeyFile(rCfg.KeyFile),
		chartmuseum.InsecureSkipVerify(rCfg.InsecureSkipVerify),
	)
	if err != nil {
		l.Error(err, "failed to create new helm push client")
		return fmt.Errorf("failed to create new helm push client : %w", err)
	}

	resp, err := c.cfg.push.UploadChartPackage(pkgPath, true)
	if err != nil {
		l.Error(err, "failed to upload chart package")
		return fmt.Errorf("failed to upload chart package with repoKey %s Name %s Version %s : %w", repoKey, chartName, version, err)
	}

	err = handlePushResponse(resp)
	defer resp.Body.Close()
	if err != nil {
		l.Error(err, "failed to handle upload/push http response")
		return fmt.Errorf("failed to handle upload/push http response for chart package with repoKey %s Name %s Version %s : %w", repoKey, chartName, version, err)
	}

	return nil
}

func handlePushResponse(resp *http.Response) error {
	if resp.StatusCode != 201 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return getChartmuseumError(b, resp.StatusCode)
	}
	return nil
}

func getChartmuseumError(b []byte, code int) error {
	var er struct {
		Error string `json:"error"`
	}
	err := json.Unmarshal(b, &er)
	if err != nil || er.Error == "" {
		return fmt.Errorf("%d: could not properly parse response JSON: %s", code, string(b))
	}
	return fmt.Errorf("%d: %s", code, er.Error)
}
