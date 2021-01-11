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

	_, err := registry.Fetch(application.Spec.HelmReleaseSpec, "", logr)
	if err != nil {
		logr.Error(err, "unable to fetch chart")
		return false, fmt.Errorf("unable to fetch the chart: %w", err)
	}

	return false, nil
}
