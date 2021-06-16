/*
Copyright 2020 The Flux authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package meta

import (
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ReadyCondition is the name of the workflow condition
	// This captures the status of the entire ApplicationGroup
	ReadyCondition string = "Ready"

	ForwardWorkflowSucceededCondition string = "ForwardWorkflowSucceeded"

	ReverseWorkflowSucceededCondition string = "ReverseWorkflowSucceeded"

	RollbackWorkflowSucceededCondition string = "RollbackWorkflowSucceeded"
)

const (
	// SucceededReason represents the condition succeeding
	SucceededReason string = "Succeeded"

	// FailedReason represents the fact that the the reconciliation failed
	FailedReason string = "Failed"

	// ProgressingReason represents the fact that the application group reconciler
	// is reconciling the app group and the forward workflow has not completed
	ProgressingReason string = "Progressing"

	// TerminatingReason represents that the application group is deleting
	// and waiting for the reverse workflow to complete
	TerminatingReason string = "Terminating"

	// SuspendedReason represents that the workflow is in a suspended state
	SuspendedReason string = "Suspended"

	// ChartPullFailedReason represents the fact that the application group reconcile
	// was unable to pull from the chart repo specified
	ChartPullFailedReason string = "ChartPullFailed"

	// WorkflowFailedReason represents the fact that a workflow step failed and is the reason
	// why the application group was unable to successfully reconcile
	WorkflowFailedReason string = "WorkflowFailed"

	// WorkflowTemplateGenerationFailedReason represents the fact that the application group was unable
	// to generate the templates for the workflow reconciliation
	WorkflowTemplateGenerationFailedReason string = "WorkflowTemplateGenerationFailed"
)

// ObjectWithStatusConditions is an interface that describes kubernetes resource
// type structs with Status Conditions
// +k8s:deepcopy-gen=false
type ObjectWithStatusConditions interface {
	GetStatusConditions() *[]metav1.Condition
}

// SetResourceCondition sets the given condition with the given status,
// reason and message on a resource.
func SetResourceCondition(obj ObjectWithStatusConditions, condition string, status metav1.ConditionStatus, reason, message string) {
	conditions := obj.GetStatusConditions()
	newCondition := metav1.Condition{
		Type:    condition,
		Status:  status,
		Reason:  reason,
		Message: message,
	}
	apimeta.SetStatusCondition(conditions, newCondition)
}

func GetResourceCondition(obj ObjectWithStatusConditions, condition string) *metav1.Condition {
	conditions := obj.GetStatusConditions()
	return apimeta.FindStatusCondition(*conditions, condition)
}

func IsFailedHelmReason(reason string) bool {
	switch reason {
	case fluxhelmv2beta1.InstallFailedReason, fluxhelmv2beta1.UpgradeFailedReason, fluxhelmv2beta1.UninstallFailedReason,
		fluxhelmv2beta1.ArtifactFailedReason, fluxhelmv2beta1.InitFailedReason, fluxhelmv2beta1.GetLastReleaseFailedReason:
		return true
	}
	return false
}
