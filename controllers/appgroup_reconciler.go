package controllers

import (
	"context"
	"os"

	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
)

var (
	ErrInvalidSpec = fmt.Errorf("custom resource spec is invalid")
	// ErrRequeue describes error while requeuing
	ErrRequeue = fmt.Errorf("(transitory error) Requeue-ing resource to try again")
)

func (r *ApplicationGroupReconciler) reconcile(ctx context.Context, l logr.Logger, ns string, appGroup *orkestrav1alpha1.ApplicationGroup) (bool, error) {
	l = l.WithValues(appgroupNameKey, appGroup.Name)
	l.V(3).Info("Reconciling ApplicationGroup object")

	if len(appGroup.Spec.Applications) == 0 {
		l.Error(ErrInvalidSpec, "ApplicationGroup must list atleast one Application")
		return false, fmt.Errorf("application group must list atleast one Application : %w", ErrInvalidSpec)
	}

	// Readiness matrix stores the status of the Application object
	// It assists in accounting for the status of all Applications in the app group
	appReadinessMatrix := populateReadinessMatrix(appGroup.Spec.Applications)
	// Cache the application objects to pass into the generate function for workflow gen
	appObjCache := make([]*orkestrav1alpha1.Application, 0, len(appGroup.Spec.Applications))

	// Lookup each application in the App group to check if status is ready or errored
	for _, application := range appGroup.Spec.Applications {
		ll := l.WithValues("application", application.Name)
		ll.V(3).Info("Looking up Application instance")

		obj := &orkestrav1alpha1.Application{}
		err := r.Client.Get(ctx, types.NamespacedName{Namespace: "", Name: application.Name}, obj)
		if err != nil {
			// requeue for IsNotFound error but for any other error this is unrecoverable
			// and should not be requeued
			if errors.IsNotFound(err) {
				ll.V(2).Info("object not found - requeueing")
				return true, nil
			}

			ll.Error(err, "unrecoverable application object GET error - will not requeue")
			return false, err
		}
		status := obj.Status

		if !status.Application.Ready || status.Application.Error != "" {
			ll.V(1).Info("application not in Ready state or in Error state - requeueing")
			return true, nil
		}

		appObjCache = append(appObjCache, obj)
		appReadinessMatrix[obj.Name] = true

		if appGroup.Status.Applications == nil {
			appGroup.Status.Applications = make([]orkestrav1alpha1.ApplicationStatus, 0)
		}
		if !entryExists(obj.Name, appGroup.Status.Applications) {
			appStatus := orkestrav1alpha1.ApplicationStatus{
				Name:        obj.Name,
				Application: obj.Status.Application,
				Subcharts:   obj.Status.Subcharts,
			}
			appGroup.Status.Applications = append(appGroup.Status.Applications, appStatus)
		}
	}

	// Safety check that all applications are accounted for and that all are in READY state
	for k, v := range appReadinessMatrix {
		if v == false {
			l.V(1).Info("application not in ready state - requeueing", "application", k)
			return true, nil
		}
	}

	// if target workflow namespace is unset, then set it to the default namespace explicitly
	if ns == "" {
		ns = defaultNamespace()
	}

	// Generate the Workflow object to submit to Argo
	return r.generateWorkflow(ctx, l, ns, appGroup, appObjCache)
}

func populateReadinessMatrix(apps []orkestrav1alpha1.DAG) map[string]bool {
	readyMatrix := make(map[string]bool)
	for _, app := range apps {
		readyMatrix[app.Name] = false
	}
	return readyMatrix
}

func entryExists(name string, ss []orkestrav1alpha1.ApplicationStatus) bool {
	for _, v := range ss {
		if v.Name == name {
			return true
		}
	}
	return false
}

func (r *ApplicationGroupReconciler) generateWorkflow(ctx context.Context, logr logr.Logger, ns string, g *orkestrav1alpha1.ApplicationGroup, apps []*orkestrav1alpha1.Application) (requeue bool, err error) {
	err = r.Engine.Generate(ctx, logr, ns, g, apps)
	if err != nil {
		logr.Error(err, "engine failed to generate workflow")
		return false, fmt.Errorf("failed to generate workflow : %w", err)
	}

	err = r.Engine.Submit(ctx, logr, g)
	if err != nil {
		logr.Error(err, "engine failed to submit workflow")
		return false, err
	}

	return false, nil
}

func defaultNamespace() string {
	if ns, ok := os.LookupEnv("WORKFLOW_NAMESPACE"); ok {
		return ns
	}
	return "orkestra"
}
