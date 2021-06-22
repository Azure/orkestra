package testutils

import (
	"context"
	"testing"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/meta"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ApplyObjToK8s saves appGroup object in the Kubernetes cluster, and
// registers cleanup functions.
func ApplyObjToK8sAndRegisterCleanup(ctx context.Context, t *testing.T, k8sClient client.Client, appGroup *v1alpha1.ApplicationGroup) {
	// Call delete on the HelmReleases for cleanup
	t.Cleanup(func() {
		_ = k8sClient.DeleteAllOf(ctx, &fluxhelmv2beta1.HelmRelease{}, client.InNamespace(appGroup.Name))
		_ = k8sClient.DeleteAllOf(ctx, &v1alpha13.Workflow{}, client.InNamespace(appGroup.Name))
	})

	// Apply the appGroup object to the cluster
	err := k8sClient.Create(ctx, appGroup)
	if err != nil {
		t.Errorf("error applying the AppGroup object to the cluster, error = %v", err)
	}

	// cleanup so that we delete the appGroup after creation
	t.Cleanup(func() {
		patch := client.MergeFrom(appGroup.DeepCopy())
		controllerutil.RemoveFinalizer(appGroup, v1alpha1.AppGroupFinalizer)
		_ = k8sClient.Patch(ctx, appGroup, patch)
		_ = k8sClient.Delete(ctx, appGroup)
	})
}

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
