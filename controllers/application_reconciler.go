package controllers

import (
	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/go-logr/logr"
)

func (r *ApplicationReconciler) reconcile(l logr.Logger, application *orkestrav1alpha1.Application) (bool, error) {
	logr := l.WithValues("application", application.Name, "group", application.Spec.GroupName)
	if application.Status.ChartStatus.Ready {
		logr.Info("application already in ready state")
		return false, nil
	}

	return false, nil
}
