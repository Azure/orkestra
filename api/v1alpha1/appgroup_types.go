// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package v1alpha1

import (
	"github.com/Azure/Orkestra/pkg/meta"
	helmopv1 "github.com/fluxcd/helm-operator/pkg/apis/helm.fluxcd.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	// Chart holds the values needed to pull the chart
	// +required
	Chart *ChartRef `json:"chart"`

	// Release holds the values to apply to the helm release
	// +required
	Release *Release `json:"release"`

	// Subcharts provides the dependency order among the subcharts of the application
	// +optional
	Subcharts []DAG `json:"subcharts,omitempty"`
}

type Release struct {
	// HelmVersion is the version of Helm to target. If not supplied,
	// the lowest _enabled Helm version_ will be targeted.
	// +kubebuilder:validation:Enum=v2;v3
	// +kubebuilder:default:=v3
	// +optional
	HelmVersion string `json:"helmVersion"`

	// Force will mark this Helm release to `--force` upgrades. This
	// forces the resource updates through delete/recreate if needed.
	// +optional
	ForceUpgrade bool `json:"forceUpgrade,omitempty"`

	// Wait will mark this Helm release to wait until all Pods,
	// PVCs, Services, and minimum number of Pods of a Deployment,
	// StatefulSet, or ReplicaSet are in a ready state before marking
	// the release as successful.
	// +optional
	Wait *bool `json:"wait,omitempty"`

	// TargetNamespace overrides the targeted namespace for the Helm
	// release. The default namespace equals to the namespace of the
	// HelmRelease resource.
	// +required
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// Timeout is the time to wait for any individual Kubernetes
	// operation (like Jobs for hooks) during installation and
	// upgrade operations.
	// +optional
	Timeout *int64 `json:"timeout,omitempty"`

	// Values holds the values for this Helm release.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:XPreserveUnknownFields
	Values helmopv1.HelmValues `json:"values,omitempty"`
}

type ChartRef struct {
	// GitChartSource references the chart repository if the
	// the helm chart is stored in a git repo
	// +optional
	helmopv1.GitChartSource `json:",inline"`

	// RepoChartSource references the chart repository of
	// a normal helm chart repo
	// +optional
	helmopv1.RepoChartSource `json:",inline"`

	// AuthSecretRef is a reference to the auth secret
	// to access a private helm repository
	// +optional
	AuthSecretRef *corev1.ObjectReference `json:"authSecretRef,omitempty"`
}

// ChartStatus shows the current status of the Application Reconciliation process
type ChartStatus struct {
	// Phase reflects the current state of the HelmRelease
	// +optional
	Phase helmopv1.HelmReleasePhase `json:"phase,omitempty"`

	// Error string from the error during reconciliation (if any)
	// +optional
	Error string `json:"error,omitempty"`

	// Version of the chart/subchart
	// +optional
	Version string `json:"version,omitempty"`

	// Staged if true denotes that the chart/subchart has been pushed to the
	// staging helm repo
	// +optional
	Staged bool `json:"staged,omitempty"`

	// +optional
	// Conditions holds the conditions for the ChartStatus
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ApplicationGroupSpec defines the desired state of ApplicationGroup
type ApplicationGroupSpec struct {
	// Applications that make up the application group
	Applications []Application `json:"applications,omitempty"`
}

// Application spec and dependency on other applications
type Application struct {
	// DAG contains the dependency information
	DAG `json:",inline"`
	// Spec contains the application spec including the chart info and overlay values
	Spec ApplicationSpec `json:"spec,omitempty"`
}

// DAG contains the dependency information
type DAG struct {
	// Name of the application
	// +required
	Name string `json:"name,omitempty"`

	// Namespace of the application
	// +required
	Namespace string `json:"namespace,omitempty"`

	// Dependencies on other applications by name
	// +optional
	Dependencies []string `json:"dependencies,omitempty"`
}

// ApplicationStatus shows the current status of the application helm release
type ApplicationStatus struct {
	// Name of the application
	// +optional
	Name string `json:"name"`

	// ChartStatus for the application helm chart
	// +optional
	ChartStatus `json:",inline"`

	// Subcharts contains the subchart chart status
	// +optional
	Subcharts map[string]ChartStatus `json:"subcharts,omitempty"`
}

// ReconciliationPhase is an enum
type ReconciliationPhase string

const (
	Init      ReconciliationPhase = "Init"
	Running   ReconciliationPhase = "Running"
	Succeeded ReconciliationPhase = "Succeeded"
	Error     ReconciliationPhase = "Error"
	Rollback  ReconciliationPhase = "Rollback"
)

// ApplicationGroupStatus defines the observed state of ApplicationGroup
type ApplicationGroupStatus struct {
	// Applications status
	// +optional
	Applications []ApplicationStatus `json:"status,omitempty"`

	// Phase is the reconciliation phase
	// +optional
	Update bool `json:"update,omitempty"`

	// ObservedGeneration captures the last generation
	// that was captured and completed by the reconciler
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions holds the conditions of the ApplicationGroup
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Progressing resets the conditions of the ApplicationGroup to
// metav1.Condition of type meta.ReadyCondition with status 'Unknown' and
// meta.StartingReason reason and message.
func (in *ApplicationGroup) Progressing() {
	in.Status.ObservedGeneration = in.Generation
	in.Status.Conditions = []metav1.Condition{}
	meta.SetResourceCondition(in, meta.ReadyCondition, metav1.ConditionUnknown, meta.ProgressingReason, "workflow is reconciling...")
	meta.SetResourceCondition(in, meta.DeployCondition, metav1.ConditionUnknown, meta.ProgressingReason, "application group is reconciling...")
}

// RollingBack sets the meta.ReadyCondition to 'True' and
// meta.RollingBack reason and message
func (in *ApplicationGroup) RollingBack() {
	meta.SetResourceCondition(in, meta.ReadyCondition, metav1.ConditionTrue, meta.FailedReason, "workflow failed because of helmreleases, rolling back...")
	meta.SetResourceCondition(in, meta.DeployCondition, metav1.ConditionTrue, meta.RollingBackReason, "rolling back because of failed helm releases...")
}

// Succeeded sets the meta.ReadyCondition to 'True', with the given
// meta.Succeeded reason and message
func (in *ApplicationGroup) ReadySucceeded() {
	meta.SetResourceCondition(in, meta.ReadyCondition, metav1.ConditionTrue, meta.SucceededReason, "workflow and reconciliation succeeded")
}

// Failed sets the meta.ReadyCondition to 'True' and
// meta.FailedReason reason and message
func (in *ApplicationGroup) ReadyFailed(message string) {
	meta.SetResourceCondition(in, meta.ReadyCondition, metav1.ConditionTrue, meta.FailedReason, message)
}

// Succeeded sets the meta.DeployCondition to 'True', with the given
// meta.Succeeded reason and message
func (in *ApplicationGroup) DeploySucceeded() {
	meta.SetResourceCondition(in, meta.DeployCondition, metav1.ConditionTrue, meta.SucceededReason, "application group reconciliation succeeded")
}

// Failed sets the meta.DeployCondition to 'True' and
// meta.FailedReason reason and message
func (in *ApplicationGroup) DeployFailed(message string) {
	meta.SetResourceCondition(in, meta.DeployCondition, metav1.ConditionTrue, meta.FailedReason, message)
}

// GetReadyCondition gets the string condition.Reason of the
// meta.ReadyCondition type
func (in *ApplicationGroup) GetReadyCondition() string {
	condition := meta.GetResourceCondition(in, meta.ReadyCondition)
	if condition == nil {
		return meta.ProgressingReason
	}
	return condition.Reason
}

// GetDeployCondition gets the string condition.Reason of the
// meta.ReadyCondition type
func (in *ApplicationGroup) GetDeployCondition() string {
	condition := meta.GetResourceCondition(in, meta.DeployCondition)
	if condition == nil {
		return meta.ProgressingReason
	}
	return condition.Reason
}

// GetStatusConditions gets the status conditions from the
// ApplicationGroup status
func (in *ApplicationGroup) GetStatusConditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=applicationgroups,scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Deploy",type="string",JSONPath=".status.conditions[?(@.type==\"Deploy\")].reason"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ApplicationGroup is the Schema for the applicationgroups API
type ApplicationGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationGroupSpec   `json:"spec,omitempty"`
	Status ApplicationGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationGroupList contains a list of ApplicationGroup
type ApplicationGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApplicationGroup{}, &ApplicationGroupList{})
}
