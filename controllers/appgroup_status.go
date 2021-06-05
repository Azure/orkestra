package controllers

import (
	"context"
	"fmt"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/workflow"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ApplicationGroupReconciler) UpdateStatusWithWorkflow(ctx context.Context, instance *v1alpha1.ApplicationGroup, patch client.Patch) (ctrl.Result, error) {
	workflowStatus, err := r.getWorkflowStatus(ctx, instance)
	if err != nil {
		return r.Failed(ctx, instance, patch, err)
	}
	requeueTime := v1alpha1.DefaultProgressingRequeue

	switch workflowStatus {
	case v1alpha12.NodeError, v1alpha12.NodeFailed:
		r.Log.V(1).Info("workflow failed with: %v", err)
		return r.Remediate(ctx, instance, patch, err)
	case v1alpha12.NodeSucceeded:
		if _, err := r.Succeeded(ctx, instance, patch); err != nil {
			return ctrl.Result{}, err
		}
		r.Log.V(1).Info("workflow has succeeded")
		requeueTime = v1alpha1.GetInterval(instance) // update the requeue time if we succeeded
	default:
		instance.Progressing()
		instance.DeploySucceeded()
		r.Log.V(1).Info("workflow is still progressing")
	}

	if err := r.UpdateStatus(ctx, instance, patch); err != nil {
		return r.Failed(ctx, instance, patch, fmt.Errorf("failed to patch the status while updating the workflow status with err: %v", err))
	}

	return ctrl.Result{RequeueAfter: requeueTime}, nil
}

func (r *ApplicationGroupReconciler) UpdateStatus(ctx context.Context, instance *v1alpha1.ApplicationGroup, patch client.Patch) error {
	chartConditionMap, subChartConditionMap, err := r.marshallChartStatus(ctx, instance)
	if err != nil {
		return err
	}
	instance.Status.Applications = getAppStatus(instance, chartConditionMap, subChartConditionMap)

	if err := r.Status().Patch(ctx, instance, patch); err != nil {
		r.Log.V(1).Error(err, "failed to patch the application group status")
		return err
	}
	return nil
}

func (r *ApplicationGroupReconciler) Failed(ctx context.Context, instance *v1alpha1.ApplicationGroup, patch client.Patch, err error) (ctrl.Result, error) {
	instance.ReadyFailed(err.Error())
	instance.DeployFailed(err.Error())
	if err := r.Status().Patch(ctx, instance, patch); err != nil {
		r.Log.V(1).Error(err, "failed to patch the application group status")
		return ctrl.Result{}, err
	}
	r.Recorder.Event(instance, "Warning", "ReconcileError", fmt.Sprintf("Failed to reconcile ApplicationGroup %v with Error : %v", instance.Name, err))
	return ctrl.Result{}, fmt.Errorf("failed to install the workflow with: %v", err)
}

func (r *ApplicationGroupReconciler) Succeeded(ctx context.Context, instance *v1alpha1.ApplicationGroup, patch client.Patch) (ctrl.Result, error) {
	// Set the status conditions into a succeeding state
	instance.ReadySucceeded()
	instance.DeploySucceeded()
	instance.Status.LastSucceededGeneration = instance.Generation
	if err := r.Status().Patch(ctx, instance, patch); err != nil {
		r.Log.V(1).Error(err, "failed to patch the application group status conditions")
		return ctrl.Result{}, err
	}

	// Set the last successful annotation for rollback scenarios
	instance.SetLastSuccessful()
	if err := r.Patch(ctx, instance, patch); err != nil {
		r.Log.V(1).Error(err, "failed to patch the application group annotations")
		return ctrl.Result{}, err
	}
	r.Recorder.Event(instance, "Normal", "ReconcileSuccess", fmt.Sprintf("Successfully reconciled ApplicationGroup %s", instance.Name))
	return ctrl.Result{}, nil
}

func (r *ApplicationGroupReconciler) Remediate(ctx context.Context, instance *v1alpha1.ApplicationGroup, patch client.Patch, err error) (ctrl.Result, error) {
	if _, err := r.Failed(ctx, instance, patch, fmt.Errorf("workflow in error state, starting to remediate")); err != nil {
		return ctrl.Result{}, err
	}
	if !r.DisableRemediation {
		return r.handleRemediation(ctx, r.Log, instance, patch, err)
	}
	return ctrl.Result{}, nil
}

func InitAppStatus(appGroup *v1alpha1.ApplicationGroup) {
	// Initialize the Status fields if not already setup
	appGroup.Status.Applications = make([]v1alpha1.ApplicationStatus, 0, len(appGroup.Spec.Applications))
	for _, app := range appGroup.Spec.Applications {
		status := v1alpha1.ApplicationStatus{
			Name:        app.Name,
			ChartStatus: v1alpha1.ChartStatus{Version: app.Spec.Chart.Version},
			Subcharts:   make(map[string]v1alpha1.ChartStatus),
		}
		appGroup.Status.Applications = append(appGroup.Status.Applications, status)
	}
}

// marshallChartStatus lists all of the HelmRelease objects that were deployed and assigns
// their status to the appropriate maps corresponding to their chart of subchart.
// These statuses are used to update the application status above
func (r *ApplicationGroupReconciler) marshallChartStatus(ctx context.Context, appGroup *v1alpha1.ApplicationGroup) (
	chartConditionMap map[string][]metav1.Condition,
	subChartConditionMap map[string]map[string][]metav1.Condition,
	err error) {
	listOption := client.MatchingLabels{
		workflow.OwnershipLabel: appGroup.Name,
		workflow.HeritageLabel:  workflow.Project,
	}

	// Init the mappings
	chartConditionMap = make(map[string][]metav1.Condition)
	subChartConditionMap = make(map[string]map[string][]metav1.Condition)

	// XXX (nitishm) Not sure why this happens ???
	// Lookup all associated HelmReleases for status as well since the Workflow will not always reflect the status of the HelmRelease
	// Lookup Workflow by ownership and heritage labels
	helmReleases := fluxhelmv2beta1.HelmReleaseList{}
	err = r.List(ctx, &helmReleases, listOption)
	if err != nil {
		r.Log.Error(err, "failed to find generated HelmRelease instance")
		return nil, nil, err
	}

	for _, hr := range helmReleases.Items {
		parent := hr.Name
		if v, ok := hr.GetAnnotations()["orkestra/parent-chart"]; ok {
			// Use the parent charts name
			parent = v
		}

		// Add the associated conditions for that helm chart to the helm chart condition
		// If the helm chart is a subchart, then add that to the subchart condition
		if parent == hr.Name {
			chartConditionMap[parent] = append(chartConditionMap[parent], hr.Status.Conditions...)
		} else {
			if _, ok := subChartConditionMap[parent]; !ok {
				subChartConditionMap[parent] = make(map[string][]metav1.Condition)
			}
			subChartConditionMap[parent][hr.Spec.ReleaseName] = append(subChartConditionMap[parent][hr.Spec.ReleaseName], hr.Status.Conditions...)
		}
	}
	return chartConditionMap, subChartConditionMap, nil
}

func (r *ApplicationGroupReconciler) getWorkflowStatus(ctx context.Context, instance *v1alpha1.ApplicationGroup) (v1alpha12.NodePhase, error) {
	wfs := v1alpha12.WorkflowList{}
	listOption := client.MatchingLabels{
		workflow.OwnershipLabel: instance.Name,
		workflow.HeritageLabel:  workflow.Project,
	}
	err := r.List(ctx, &wfs, listOption)
	if err != nil {
		r.Log.Error(err, "failed to find generated workflow instance")
		return "", err
	}
	if wfs.Items.Len() == 0 {
		err = fmt.Errorf("no associated workflow found")
		r.Log.Error(err, "no associated workflow found")
		return "", err
	}
	return wfs.Items[0].Status.Phase, nil
}

func getAppStatus(appGroup *v1alpha1.ApplicationGroup, chartConditionMap map[string][]metav1.Condition,
	subChartConditionMap map[string]map[string][]metav1.Condition) []v1alpha1.ApplicationStatus {
	// Update each application status using the HelmRelease status
	var v []v1alpha1.ApplicationStatus
	for _, app := range appGroup.Status.Applications {
		app.ChartStatus.Conditions = chartConditionMap[app.Name]
		for subchartName, subchartStatus := range app.Subcharts {
			subchartStatus.Conditions = subChartConditionMap[app.Name][subchartName]
			app.Subcharts[subchartName] = subchartStatus
		}
		v = append(v, app)
	}
	return v
}
