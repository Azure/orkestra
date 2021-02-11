package controllers

import (
	"context"
	"fmt"
	"net/url"
	"os"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/chart"
)

func (r *ApplicationReconciler) reconcile(ctx context.Context, l logr.Logger, application *orkestrav1alpha1.Application) (bool, error) {
	stagingDir := r.TargetDir + "/" + r.StagingRepoName

	ll := l.WithValues("application", application.Name, "group", application.Spec.GroupID)

	application.Status.Name = application.Name

	if application.Status.Subcharts == nil {
		application.Status.Subcharts = make(map[string]orkestrav1alpha1.ChartStatus)
	}

	if application.Status.Application.Ready {
		ll.Info("application already in ready state")
		return false, nil
	}

	repoKey := application.Spec.ChartRepoNickname
	repoPath := application.Spec.RepoPath
	name := application.Spec.HelmReleaseSpec.Name
	version := application.Spec.HelmReleaseSpec.Version

	fpath, appCh, err := r.RegistryClient.PullChart(ll, repoKey, repoPath, name, version)
	defer func() {
		if r.Cfg.Cleanup {
			os.Remove(fpath)
		}
	}()
	if err != nil || appCh == nil {
		ll.Error(err, "failed to pull application chart")
		return false, fmt.Errorf("failed to pull application chart %s/%s:%s : %w", repoKey, name, version, err)
	}

	if appCh.Dependencies() != nil {
		stagingRepoName := r.StagingRepoName
		// If Dependencies - extract subchart and push each to staging registry
		if isDependenciesEmbedded(appCh) {
			for _, sc := range appCh.Dependencies() {
				cs := orkestrav1alpha1.ChartStatus{}

				if err := sc.Validate(); err != nil {
					cs.Error = err.Error()
					ll.Error(err, "failed to validate application subchart for staging registry")
					return false, fmt.Errorf("failed to validate application subchart for staging registry : %w", err)
				}

				path, err := registry.SaveChartPackage(sc, stagingDir)
				if err != nil {
					cs.Error = err.Error()
					ll.Error(err, "failed to save subchart package as tgz")
					return false, fmt.Errorf("failed to save subchart package as tgz at location %s : %w", path, err)
				}

				err = r.RegistryClient.PushChart(ll, stagingRepoName, path, sc)

				if err != nil {
					cs.Error = err.Error()
					ll.Error(err, "failed to push application subchart to staging registry")
					return false, fmt.Errorf("failed to push application subchart to staging registry : %w", err)
				}

				cs.Staged = true
				cs.Version = sc.Metadata.Version
				cs.Ready = true
				cs.Error = ""

				application.Status.Subcharts[sc.Name()] = cs
				application.Status.Application.Staged = true
			}
		} else {
			for _, sc := range appCh.Dependencies() {
				cs := orkestrav1alpha1.ChartStatus{
					Staged:  false,
					Version: sc.Metadata.Version,
					Ready:   true,
					Error:   "",
				}
				application.Status.Subcharts[sc.Name()] = cs
				application.Status.Application.Staged = false
			}
		}

		// Unset dependencies by disabling them.
		// Using appCh.SetDependencies() does not cut it since some charts rely on subcharts for tpl helpers
		// provided in the charts directory.
		// IMPORTANT: This expects charts to follow best practices to allow enabling and disabling subcharts
		// See: https://helm.sh/docs/topics/charts/ #Chart Dependencies
		for _, dep := range appCh.Metadata.Dependencies {
			// Disable subchart through metadata
			dep.Enabled = false
			// Precautionary - overrite values with subcharts disabled:93
			appCh.Values[dep.Name] = map[string]interface{}{
				"enabled": false,
			}
		}

		if err := appCh.Validate(); err != nil {
			application.Status.Application.Error = err.Error()
			ll.Error(err, "failed to validate application chart for staging registry")
			return false, fmt.Errorf("failed to validate application chart for staging registry : %w", err)
		}

		_, err := registry.SaveChartPackage(appCh, stagingDir)
		if err != nil {
			ll.Error(err, "failed to save modified app chart to filesystem")
			return false, fmt.Errorf("failed to save modified app chart to filesystem : %w", err)
		}

		// Replace existing chart with modified chart
		path := stagingDir + "/" + application.Spec.HelmReleaseSpec.Name + "-" + appCh.Metadata.Version + ".tgz"
		err = r.RegistryClient.PushChart(ll, stagingRepoName, path, appCh)
		if err != nil {
			application.Status.Application.Error = err.Error()
			ll.Error(err, "failed to push modified application chart to staging registry")
			return false, fmt.Errorf("failed to push modified application chart to staging registry : %w", err)
		}
	}

	return false, nil
}

func isDependenciesEmbedded(ch *chart.Chart) bool {
	isURI := false
	for _, d := range ch.Metadata.Dependencies {
		if _, err := url.ParseRequestURI(d.Repository); err == nil {
			isURI = true
		}
	}

	if !isURI {
		if len(ch.Dependencies()) > 0 {
			return true
		}
	}
	return false
}
