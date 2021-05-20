package utils

import (
	"os"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

func HelmUninstall(release, namespace string) error {
	os.Setenv("HELM_NAMESPACE", namespace)
	var settings = cli.New()
	settings.Debug = false

	actionConfig := new(action.Configuration)
	helmDriver := os.Getenv("HELM_DRIVER")
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), helmDriver, debug); err != nil {
		return err
	}
	client := action.NewUninstall(actionConfig)
	_, err := client.Run(release)
	if err != nil {
		return err
	}
	return nil
}

func HelmRollback(release, namespace string) error {
	os.Setenv("HELM_NAMESPACE", namespace)
	var settings = cli.New()
	settings.Debug = false

	actionConfig := new(action.Configuration)
	helmDriver := os.Getenv("HELM_DRIVER")
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), helmDriver, debug); err != nil {
		return err
	}
	client := action.NewRollback(actionConfig)
	client.Wait = true
	client.Recreate = true
	client.Timeout = time.Minute * 5
	err := client.Run(release)
	if err != nil {
		return err
	}
	return nil
}
func debug(format string, v ...interface{}) {

}
