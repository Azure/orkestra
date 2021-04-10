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
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ReadyCondition is the name of the Ready condition
	ReadyCondition string = "Ready"
)

const (
	// SucceededReason represents the fact that the reconciliation succeeded
	SucceededReason string = "Succeeded"

	// FailedReason represents the fact that the the reconciliation failed
	FailedReason string = "Failed"

	// HelmReleaseFailedReason represents the fact that the underlying HelmRelease has failed
	// so we need to perform a rollback to remediate
	HelmReleaseFailedReason = "HelmReleaseFailed"

	// RunningReason represents the fact that the workflow is in a running state
	RunningReason string = "Running"

	// StartingReason represents the fact that the workflow has been initialized and has
	// not yet reached the running state
	StartingReason string = "Starting"

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
