package controllers

import (
	"fmt"
	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/go-logr/logr"
)

func (r *ApplicationReconciler) reconcile(l logr.Logger, application *orkestrav1alpha1.Application) (bool, error) {
	logr := l.WithValues("application", application.Name, "group", application.Spec.GroupId)
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
	var artifactoryUrl string

	err = registry.PushToStaging(ch, artifactoryUrl, logr,"test","test","test")
	if err != nil {
		logr.Error(err, "unable to load chart")
		return false, fmt.Errorf("unable to push to harbour : %w", err)
	}

	return false, err
}
