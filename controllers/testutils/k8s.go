package testutils

import (
	"context"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/meta"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsWorkflowInRunningState(ctx context.Context, k8sClient client.Client, name, namespace string) bool {
	workflow := &v1alpha13.Workflow{}
	workflowKey := types.NamespacedName{Name: name, Namespace: namespace}
	_ = k8sClient.Get(ctx, workflowKey, workflow)
	return string(workflow.Status.Phase) == string(v1alpha13.NodeRunning)
}

func IsWorkflowInSuspendedState(ctx context.Context, k8sClient client.Client, name, namespace string) bool {
	workflow := &v1alpha13.Workflow{}
	workflowKey := types.NamespacedName{Name: name, Namespace: namespace}
	_ = k8sClient.Get(ctx, workflowKey, workflow)
	return workflow.Spec.Suspend != nil && *workflow.Spec.Suspend
}

func IsAppGroupInSucceededReason(ctx context.Context, k8sClient client.Client, appGroup *v1alpha1.ApplicationGroup) bool {
	key := client.ObjectKeyFromObject(appGroup)
	appGroup = &v1alpha1.ApplicationGroup{}
	if err := k8sClient.Get(ctx, key, appGroup); err != nil {
		return false
	}
	return appGroup.GetReadyCondition() == meta.SucceededReason
}

func IsAppGroupInChartPullFailedReason(ctx context.Context, k8sClient client.Client, appGroup *v1alpha1.ApplicationGroup) bool {
	key := client.ObjectKeyFromObject(appGroup)
	if err := k8sClient.Get(ctx, key, appGroup); err != nil {
		return false
	}
	readyCondition := meta.GetResourceCondition(appGroup, meta.ReadyCondition)
	if readyCondition == nil {
		return false
	}
	return readyCondition.Reason == meta.ChartPullFailedReason
}

func IsAppGroupInProgressingReason(ctx context.Context, k8sClient client.Client, appGroup *v1alpha1.ApplicationGroup) bool {
	key := client.ObjectKeyFromObject(appGroup)
	if err := k8sClient.Get(ctx, key, appGroup); err != nil {
		return false
	}
	return appGroup.GetReadyCondition() == meta.ProgressingReason
}
