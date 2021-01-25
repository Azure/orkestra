package controllers

import (
	"fmt"
	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/go-logr/logr"
)

func (r *ApplicationReconciler) reconcile(l logr.Logger, application *orkestrav1alpha1.Application) (bool, error) {
	logr := l.WithValues("application", application.Name, "group", application.Spec.GroupID)
	if application.Status.ChartStatus.Ready {
		logr.Info("application already in ready state")
		return false, nil
	}

	chartLocation, err := registry.Fetch(application.Spec.HelmReleaseSpec, "", logr)
	if err != nil {
		logr.Error(err, "unable to fetch chart")
		return false, fmt.Errorf("unable to fetch the chart: %w", err)
	}

	ch, err := registry.Load(chartLocation, false, logr)
	if err != nil {
		logr.Error(err, "unable to load chart")
		return false, fmt.Errorf("unable to load the chart: %w", err)
	}
	var artifactoryURL string

	err = registry.PushToStaging(ch, logr, "test", "test", artifactoryURL)
	if err != nil {
		logr.Error(err, "unable to load chart")
		return false, fmt.Errorf("unable to push to staging : %w", err)
	}

	return false, err
}
