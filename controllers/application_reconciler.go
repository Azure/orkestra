package controllers

import (
	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/go-logr/logr"
)

func (r *ApplicationReconciler) reconcile(l logr.Logger, application *orkestrav1alpha1.Application) (bool, error) {
	return false, nil
}
