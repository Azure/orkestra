package controllers

import (
	"context"
	"fmt"
	"net/url"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/chart"
)

func (r *ApplicationReconciler) reconcile(ctx context.Context, l logr.Logger, application *orkestrav1alpha1.Application) (bool, error) {
	ll := l.WithValues("application", application.Name, "group", application.Spec.GroupID)
	if application.Status.ChartStatus.Ready {
		ll.Info("application already in ready state")
		return false, nil
	}

	repoURL := application.Spec.HelmReleaseSpec.RepoURL
	name := application.Spec.HelmReleaseSpec.Name
	version := application.Spec.HelmReleaseSpec.Version

	appCh, err := r.RegistryClient.PullChart(ll, repoURL, name, version)
	if err != nil || appCh == nil {
		ll.Error(err, "failed to pull application chart")
		return false, fmt.Errorf("failed to pull application chart %s/%s:%s : %w", repoURL, name, version, err)
	}

	// If Dependencies - extract subchart and push each to staging registry
	if !isDependenciesEmbedded(appCh) {
		stagingRepoURL := r.StagingRepoURL

		if appCh.Dependencies() != nil {
			for _, sc := range appCh.Dependencies() {
				err := r.RegistryClient.PushChart(ll, stagingRepoURL, sc)
				if err != nil {
					ll.Error(err, "failed to push application subchart for staging registry")
					return false, fmt.Errorf("failed to validate application chart for staging registry : %w", err)
				}
			}
		}

		// unset dependencies
		appCh.SetDependencies()
		if err := appCh.Validate(); err != nil {
			ll.Error(err, "failed to validate application chart for staging registry")
			return false, fmt.Errorf("failed to validate application chart for staging registry : %w", err)
		}

		err := r.RegistryClient.PushChart(ll, stagingRepoURL, appCh)
		if err != nil {

		}
	}

	return false, err
}

func isDependenciesEmbedded(ch *chart.Chart) bool {
	for _, d := range ch.Metadata.Dependencies {
		if _, err := url.ParseRequestURI(d.Repository); err == nil {
			return false
		}
	}
	return true
}
