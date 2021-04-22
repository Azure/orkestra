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
	helmopv1 "github.com/fluxcd/helm-operator/pkg/apis/helm.fluxcd.io/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ReadyCondition is the name of the workflow condition
	// This captures the status of the entire ApplicationGroup
	ReadyCondition string = "Ready"

	// DeployCondition is the name of the Deploy condition
	// This captures the state of receiving and reacting to the spec by the reconciler
	DeployCondition string = "Deploy"

	// ReleasedCondition represents the status of the last release attempt
	// (install/upgrade/test) against the latest desired state.
	ReleasedCondition string = "Released"
)

const (
	// SucceededReason represents the fact that the reconciliation succeeded
	SucceededReason string = "Succeeded"

	// FailedReason represents the fact that the the reconciliation failed
	FailedReason string = "Failed"

	// ProgressingReason represents the fact that the workflow is in a starting
	// or running state, we have not reached a terminal state yet
	ProgressingReason string = "Progressing"

	// RollbackReason represents the fact that we are entering a rollback state
	// and is transitioning into a non-terminal state
	RollingBackReason string = "RollingBack"
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

func ToStatusConditions(conditions []helmopv1.HelmReleaseCondition) []metav1.Condition {
	var newConditions []metav1.Condition
	for _, condition := range conditions {
		newCondition := metav1.Condition{}
		newCondition.LastTransitionTime = *condition.LastTransitionTime
		newCondition.Status = metav1.ConditionStatus(condition.Status)
		newCondition.Message = condition.Message
		newCondition.Reason = condition.Reason
		newCondition.Type = string(condition.Type)
		newConditions = append(newConditions, newCondition)
	}
	return newConditions
}
