// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package v1alpha1

import (
	helmopv1 "github.com/fluxcd/helm-operator/pkg/apis/helm.fluxcd.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	// Subcharts provides the dependency order among the subcharts of the application
	Subcharts []DAG  `json:"subcharts,omitempty"`
	GroupID   string `json:"groupID,omitempty"`

	// ChartRepoNickname is used to lookup the repository config in the registries config map
	ChartRepoNickname string `json:"repo,omitempty"`

	// XXX (nitishm) **IMPORTANT**: DO NOT USE HelmReleaseSpec.Values!!!
	// ApplicationSpec.Overlays field replaces HelmReleaseSpec.Values field.
	// Setting the HelmReleaseSpec.Values field will not reflect in the deployed Application object
	//
	// Explanation
	// ===========
	// HelmValues uses a map[string]interface{} structure for holding helm values Data.
	// kubebuilder prunes the field value when deploying the Application resource as it considers the field to be an
	// Unknown field. HelmOperator v1 being in maintenance mode, we do not expect them to merge PRs
	// to add the  +kubebuilder:pruning:PreserveUnknownFields
	// https://github.com/fluxcd/helm-operator/issues/585

	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:XPreserveUnknownFields
	Overlays helmopv1.HelmValues `json:"overlays,omitempty"`

	// RepoPath provides the subdir path to the actual chart artifact within a Helm Registry
	// Artifactory for instance utilizes folders to store charts
	RepoPath string `json:"repoPath,omitempty"`

	// Inline HelmReleaseSpec from the flux helm-operator package
	helmopv1.HelmReleaseSpec `json:",inline"`
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
	Name string `json:"name,omitempty"`
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
