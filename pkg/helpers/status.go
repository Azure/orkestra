package helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/Orkestra/pkg/meta"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/workflow"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatusHelper struct {
	client.Client
	logr.Logger
	PatchFrom client.Patch
	Recorder  record.EventRecorder
}

func (helper *StatusHelper) UpdateStatusWithWorkflow(ctx context.Context, instance *v1alpha1.ApplicationGroup) (shouldRemediate bool, requeueTime time.Duration, err error) {
	workflowStatus, err := helper.getWorkflowStatus(ctx, instance.Name)
	if err != nil {
		return false, time.Duration(0), err
	}
	requeueTime = v1alpha1.DefaultProgressingRequeue

	switch string(workflowStatus.Phase) {
	case string(v1alpha13.NodeError), string(v1alpha13.NodeFailed):
		helper.Info("workflow node is in failed state")
		err = helper.Remediating(ctx, instance)
		return true, time.Duration(0), err
	case string(v1alpha13.NodeSucceeded):
		if _, err := helper.Succeeded(ctx, instance); err != nil {
			return false, time.Duration(0), err
		}
		helper.V(1).Info("workflow has succeeded")
		requeueTime = v1alpha1.GetInterval(instance) // update the requeue time if we succeeded
	default:
		if instance.GetReadyCondition() == meta.RollingBackReason {
			if err := helper.RollingBack(ctx, instance); err != nil {
				return false, time.Duration(0), err
			}
		}
		helper.V(1).Info("workflow is still progressing")
	}
	return false, requeueTime, nil
}

func (helper *StatusHelper) UpdateStatus(ctx context.Context, instance *v1alpha1.ApplicationGroup) error {
	chartConditionMap, subChartConditionMap, err := helper.marshallChartStatus(ctx, instance)
	if err != nil {
		return err
	}
	instance.Status.Applications = getAppStatus(instance, chartConditionMap, subChartConditionMap)
	instance.DeploySucceeded()

	return helper.patchStatus(ctx, instance)
}

func (helper *StatusHelper) Failed(ctx context.Context, instance *v1alpha1.ApplicationGroup, err error) (ctrl.Result, error) {
	helper.Recorder.Event(instance, "Warning", "ReconcileError", fmt.Sprintf("Failed to reconcile ApplicationGroup %v with Error : %v", instance.Name, err))
	instance.ReadyFailed(err.Error())
	instance.DeployFailed(err.Error())

	// We don't care if this call fails because we will retry anyways
	_ = helper.patchStatus(ctx, instance)

	return ctrl.Result{}, err
}

func (helper *StatusHelper) Succeeded(ctx context.Context, instance *v1alpha1.ApplicationGroup) (ctrl.Result, error) {
	// Set the status conditions into a succeeding state
	instance.ReadySucceeded()
	instance.DeploySucceeded()
	if err := helper.patchStatus(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	// Set the last successful annotation for rollback scenarios
	instance.SetLastSuccessful()
	if err := helper.Patch(ctx, instance, helper.PatchFrom); err != nil {
		helper.V(1).Error(err, "failed to patch the application group annotations")
		return ctrl.Result{}, err
	}
	helper.Recorder.Event(instance, "Normal", "ReconcileSuccess", fmt.Sprintf("Successfully reconciled ApplicationGroup %s", instance.Name))
	return ctrl.Result{}, nil
}

func (helper *StatusHelper) Remediating(ctx context.Context, instance *v1alpha1.ApplicationGroup) error {
	instance.ReadyFailed("workflow in failed state, starting rollback...")
	instance.DeployFailed("workflow in failed state, starting rollback...")

	return helper.patchStatus(ctx, instance)
}

// RollingBack sets the meta.ReadyCondition to 'True' and
// meta.RollingBack reason and message
func (helper *StatusHelper) RollingBack(ctx context.Context, instance *v1alpha1.ApplicationGroup) error {
	meta.SetResourceCondition(instance, meta.ReadyCondition, metav1.ConditionTrue, meta.FailedReason, "rolling back because of failed workflow during upgrade...")
	meta.SetResourceCondition(instance, meta.DeployCondition, metav1.ConditionTrue, meta.RollingBackReason, "rolling back because of failed workflow during upgrade...")

	return helper.patchStatus(ctx, instance)
}

// Progressing resets the conditions of the ApplicationGroup to
// metav1.Condition of type meta.ReadyCondition with status 'Unknown' and
// meta.StartingReason reason and message.
func (helper *StatusHelper) Progressing(ctx context.Context, instance *v1alpha1.ApplicationGroup) error {
	instance.Status.Conditions = []metav1.Condition{}
	meta.SetResourceCondition(instance, meta.ReadyCondition, metav1.ConditionUnknown, meta.ProgressingReason, "workflow is reconciling...")
	meta.SetResourceCondition(instance, meta.DeployCondition, metav1.ConditionUnknown, meta.ProgressingReason, "application group is reconciling...")

	return helper.patchStatus(ctx, instance)
}

// MarkReversing sets the meta.ReadyCondition to 'False', with the given
// meta.Reversing reason and message
func (helper *StatusHelper) MarkReversing(ctx context.Context, instance *v1alpha1.ApplicationGroup) error {
	meta.SetResourceCondition(instance, meta.ReadyCondition, metav1.ConditionFalse, meta.ReversingReason, "application group is reversing...")
	return helper.patchStatus(ctx, instance)
}

func initAppStatus(appGroup *v1alpha1.ApplicationGroup) {
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

func (helper *StatusHelper) getWorkflowStatus(ctx context.Context, appGroupName string) (*v1alpha13.WorkflowStatus, error) {
	wfs := v1alpha13.WorkflowList{}
	listOption := client.MatchingLabels{
		workflow.OwnershipLabel: appGroupName,
		workflow.HeritageLabel:  workflow.Project,
	}
	err := helper.List(ctx, &wfs, listOption)
	if err != nil {
		helper.Error(err, "failed to find generated workflow instance")
		return nil, err
	}
	if wfs.Items.Len() == 0 {
		err = fmt.Errorf("no associated workflow found")
		helper.Error(err, "no associated workflow found")
		return nil, err
	}
	return &wfs.Items[0].Status, nil
}

// marshallChartStatus lists all of the HelmRelease objects that were deployed and assigns
// their status to the appropriate maps corresponding to their chart of subchart.
// These statuses are used to update the application status above
func (helper *StatusHelper) marshallChartStatus(ctx context.Context, appGroup *v1alpha1.ApplicationGroup) (
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

	// Lookup all associated HelmReleases for status as well since the Workflow will not always reflect the status of the HelmRelease
	// Lookup Workflow by ownership and heritage labels
	helmReleases := fluxhelmv2beta1.HelmReleaseList{}
	err = helper.List(ctx, &helmReleases, listOption)
	if err != nil {
		helper.Error(err, "failed to find generated HelmRelease instance")
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

func (helper *StatusHelper) patchStatus(ctx context.Context, instance *v1alpha1.ApplicationGroup) error {
	if err := helper.Status().Patch(ctx, instance, helper.PatchFrom); err != nil {
		helper.V(1).Error(err, "failed to patch the application group status")
		return err
	}
	return nil
}
