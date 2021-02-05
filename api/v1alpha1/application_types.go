// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package v1alpha1

import (
	helmopv1 "github.com/fluxcd/helm-operator/pkg/apis/helm.fluxcd.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	// Namespace to which the HelmRelease object will be deployed
	Namespace string `json:"namespace,omitempty"`
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
	// Unknown field. HelmOperator v1 being in maintainence mode, we do not expect them to merge PRs
	// to add the  +kubebuilder:pruning:PreserveUnknownFields
	// https://github.com/fluxcd/helm-operator/issues/585

	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:XPreserveUnknownFields
	Overlays                 helmopv1.HelmValues `json:"overlays,omitempty"`
	helmopv1.HelmReleaseSpec `json:",inline"`
}

// ChartStatus denotes the current status of the Application Reconciliation
type ChartStatus struct {
	Ready   bool   `json:"ready,omitempty"`
	Error   string `json:"error,omitempty"`
	Staged  bool   `json:"staged,omitempty"`
	Version string `json:"version,omitempty"`
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
	Name        string                 `json:"name"`
	Application ChartStatus            `json:"application"`
	Subcharts   map[string]ChartStatus `json:"subcharts"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=applications,scope=Cluster
// +kubebuilder:subresource:status

// Application is the Schema for the applications API
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
