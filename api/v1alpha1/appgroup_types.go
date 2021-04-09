// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package v1alpha1

import (
	helmopv1 "github.com/fluxcd/helm-operator/pkg/apis/helm.fluxcd.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	// Subcharts provides the dependency order among the subcharts of the application
	// +optional
	Subcharts []DAG `json:"subcharts,omitempty"`

	// +required
	GroupID string `json:"groupId,omitempty"`

	// +required
	Chart *ChartRef `json:"chart"`

	// +required
	Release *Release `json:",inline"`

	// Values holds the values for this Helm release.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:XPreserveUnknownFields
	Values helmopv1.HelmValues `json:"values,omitempty"`
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
	// +optional
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// Timeout is the time to wait for any individual Kubernetes
	// operation (like Jobs for hooks) during installation and
	// upgrade operations.
	// +optional
	Timeout *int64 `json:"timeout,omitempty"`
}

type ChartRef struct {
	// +optional
	helmopv1.GitChartSource `json:",inline"`

	// +optional
	helmopv1.RepoChartSource `json:",inline"`

	// +optional
	HelmRepoSecretRef *corev1.ObjectReference `json:"helmRepoSecretRef,omitempty"`
}

// ChartStatus shows the current status of the Application Reconciliation process
type ChartStatus struct {
	// Phase reflects the current state of the HelmRelease
	Phase helmopv1.HelmReleasePhase `json:"phase,omitempty"`
	// Error string from the error during reconciliation (if any)
	Error string `json:"error,omitempty"`
	// Version of the chart/subchart
	Version string `json:"version,omitempty"`
	// Staged if true denotes that the chart/subchart has been pushed to the
	// staging helm repo
	Staged bool `json:"staged,omitempty"`
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
	Dependencies []string `json:"dependencies,omitempty"`
}

// ApplicationStatus shows the current status of the application helm release
type ApplicationStatus struct {
	// Name of the application
	Name string `json:"name"`
	// ChartStatus for the application helm chart
	ChartStatus `json:",inline"`
	// Subcharts contains the subchart chart status
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
	// Checksums for each application are calculated from the application spec
	// The status/metadata information is ignored
	Checksums map[string]string `json:"checksums,omitempty"`
	// Applications status
	Applications []ApplicationStatus `json:"status,omitempty"`
	// Phase is the reconciliation phase
	Phase ReconciliationPhase `json:"phase,omitempty"`
	// Update is an internal flag used to trigger a workflow update
	Update bool `json:"update,omitempty"`
	// Error string from errors during reconciliation
	Error string `json:"error,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=applicationgroups,scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=`.metadata.creationTimestamp`
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
