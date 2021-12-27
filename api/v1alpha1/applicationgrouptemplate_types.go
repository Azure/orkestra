// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplicationGroupTemplateSpec defines the desired state of ApplicationGroupTemplate
type ApplicationGroupTemplateSpec struct {
	// ApplicationGroupTemplateSpec defines the desired state of ApplicationGroupTemplate
	// +required
	Template ApplicationGroupSpec `json:"template,omitempty"`
}

//+kubebuilder:object:root=true

// ApplicationGroupTemplate is the Schema for the applicationgrouptemplates API
type ApplicationGroupTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ApplicationGroupTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=applicationgrouptemplates,scope=Cluster,shortName={"agt","appgrouptpl"}
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ApplicationGroupTemplateList contains a list of ApplicationGroupTemplate
type ApplicationGroupTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationGroupTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApplicationGroupTemplate{}, &ApplicationGroupTemplateList{})
}
