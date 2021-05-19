// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package v1alpha1

import (
	"encoding/json"
	"time"

	"github.com/Azure/Orkestra/pkg/meta"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultProgressingRequeue = 5 * time.Second
	DefaultSucceededRequeue   = 5 * time.Minute
)

// GetInterval returns the interval if specified in the application group
// Otherwise, it returns the default requeue time for the appGroup
func GetInterval(appGroup *ApplicationGroup) time.Duration {
	if appGroup.Spec.Interval != nil {
		return appGroup.Spec.Interval.Duration
	}
	return DefaultSucceededRequeue
}

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
	// Interval at which to reconcile the Helm release.
	// +kubebuilder:default:="5m"
	// +optional
	Interval metav1.Duration `json:"interval,omitempty"`

	// TargetNamespace to target when performing operations for the HelmRelease.
	// Defaults to the namespace of the HelmRelease.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Optional
	// +optional
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// Timeout is the time to wait for any individual Kubernetes operation (like Jobs
	// for hooks) during the performance of a Helm action. Defaults to '5m0s'.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Values holds the values for this Helm release.
	// +optional
	Values *apiextensionsv1.JSON `json:"values,omitempty"`

	// Install holds the configuration for Helm install actions for this HelmRelease.
	// +optional
	Install *fluxhelmv2beta1.Install `json:"install,omitempty"`

	// Upgrade holds the configuration for Helm upgrade actions for this HelmRelease.
	// +optional
	Upgrade *fluxhelmv2beta1.Upgrade `json:"upgrade,omitempty"`

	// Rollback holds the configuration for Helm rollback actions for this HelmRelease.
	// +optional
	Rollback *fluxhelmv2beta1.Rollback `json:"rollback,omitempty"`

	// Rollback holds the configuration for Helm uninstall actions for this HelmRelease.
	// +optional
	Uninstall *fluxhelmv2beta1.Uninstall `json:"uninstall,omitempty"`
}

type ChartRef struct {
	// The Helm repository URL, a valid URL contains at least a protocol and host.
	// +required
	Url string `json:"url"` //nolint: golint

	// The name or path the Helm chart is available at in the SourceRef.
	// +required
	Name string `json:"name"` //nolint: golint

	// Version semver expression, ignored for charts from v1beta1.GitRepository and
	// v1beta1.Bucket sources. Defaults to latest when omitted.
	// +kubebuilder:default:=*
	// +optional
	Version string `json:"version,omitempty"`

	// AuthSecretRef is a reference to the auth secret
	// to access a private helm repository
	// +optional
	AuthSecretRef *corev1.ObjectReference `json:"authSecretRef,omitempty"`
}

// ChartStatus shows the current status of the Application Reconciliation process
type ChartStatus struct {
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

	// Interval specifies the between reconciliations of the ApplicationGroup
	// Defaults to 5s for short requeue and 30s for long requeue
	// +optional
	Interval *metav1.Duration `json:"interval,omitempty"`
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
	Name string `json:"name"`

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

// GetValues unmarshals the raw values to a map[string]interface{} and returns
// the result.
func (in *Application) GetValues() map[string]interface{} {
	var values map[string]interface{}
	if in.Spec.Release.Values != nil {
		_ = json.Unmarshal(in.Spec.Release.Values.Raw, &values)
	}
	return values
}

func GetJSON(values map[string]interface{}) (*apiextensionsv1.JSON, error) {
	bytes, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	return &apiextensionsv1.JSON{
		Raw: bytes,
	}, nil
}

// SetValues marshals the raw values into the JSON values
func (in *Application) SetValues(values map[string]interface{}) error {
	bytes, err := json.Marshal(values)
	if err != nil {
		return err
	}
	in.Spec.Release.Values.Raw = bytes
	return nil
}

// Progressing resets the conditions of the ApplicationGroup to
// metav1.Condition of type meta.ReadyCondition with status 'Unknown' and
// meta.StartingReason reason and message.
func (in *ApplicationGroup) Progressing() {
	in.Status.Conditions = []metav1.Condition{}
	meta.SetResourceCondition(in, meta.ReadyCondition, metav1.ConditionUnknown, meta.ProgressingReason, "workflow is reconciling...")
	meta.SetResourceCondition(in, meta.DeployCondition, metav1.ConditionUnknown, meta.ProgressingReason, "application group is reconciling...")
}

// RollingBack sets the meta.ReadyCondition to 'True' and
// meta.RollingBack reason and message
func (in *ApplicationGroup) RollingBack() {
	meta.SetResourceCondition(in, meta.ReadyCondition, metav1.ConditionTrue, meta.FailedReason, "workflow failed because of helmreleases during upgrade, rolling back...")
	meta.SetResourceCondition(in, meta.DeployCondition, metav1.ConditionTrue, meta.RollingBackReason, "rolling back because of failed helm releases...")
}

// Reversing sets the meta.ReadyCondition to 'False', with the given
// meta.Reversing reason and message
func (in *ApplicationGroup) Reversing() {
	meta.SetResourceCondition(in, meta.ReadyCondition, metav1.ConditionFalse, meta.ReversingReason, "workflow reversing because of helmreleases during install...")
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

// GetStatusConditions gets the status conditions from the
// ChartStatus status
func (in *ChartStatus) GetStatusConditions() *[]metav1.Condition {
	return &in.Conditions
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=applicationgroups,scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Deploy",type="string",JSONPath=".status.conditions[?(@.type==\"Deploy\")].reason"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ApplicationGroup is the Schema for the applicationgroups API
type ApplicationGroup struct { //nolint: gocritic
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

func (a *Release) DisableWaitForInstall() bool {
	if a != nil && a.Install != nil {
		return a.Install.DisableWait
	}
	return false
}

func (a *Release) DisableWaitForUpgrade() bool {
	if a != nil && a.Upgrade != nil {
		return a.Upgrade.DisableWait
	}
	return false
}

func (a *Release) DisableWaitForRollback() bool {
	if a != nil && a.Rollback != nil {
		return a.Rollback.DisableWait
	}
	return false
}

func (a *Release) ForceForUpgrade() bool {
	if a != nil && a.Upgrade != nil {
		return a.Upgrade.Force
	}
	return false
}
