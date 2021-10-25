// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineSpec defines the desired state of Pipeline
type PipelineSpec struct {
	// Preflight is a list of preflight checks to run before the pipeline runs.
	// +optional
	PreFlight []PipelineTask `json:"preFlight,omitempty"`

	// ApplicationGroup is the name of the application group to run the pipeline in.
	// +required
	ApplicationGroup corev1.ObjectReference `json:"applicationGroup,omitempty"`

	// Postflight is a list of postflight checks to run after the pipeline runs.
	// +optional
	PostFlight []PipelineTask `json:"postFlight,omitempty"`
}

type PipelineTask struct {
	// DAG contains the dependency information
	// +required
	DAG `json:",inline"`

	// Container is the container to run the task in.
	// +required
	Container corev1.Container `json:"container,omitempty"`
}

// PipelineStatus defines the observed state of Pipeline
type PipelineStatus struct {
}

//+kubebuilder:object:root=true
// +kubebuilder:resource:path=pipelines,scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Pipeline is the Schema for the pipelines API
type Pipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineSpec   `json:"spec,omitempty"`
	Status PipelineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PipelineList contains a list of Pipeline
type PipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Pipeline{}, &PipelineList{})
}
