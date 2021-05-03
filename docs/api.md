---
layout: default
title: API Reference 
nav_order: 3
---
# API Reference

## Spec

### ApplicationGroupSpec

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| Applications | [ ][`Application`](#application) | Applications that make up the application group | Yes | |
| Interval | [`*metav1.Duration`](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration) | Interval specifies the interval between reconciliations of the `ApplicationGroup` | No | **5s, 30s** (*short requeue interval, long requeue on Success*) |

### Application

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| *inline* | [`DAG`](#dag) | Dependencies between the Applications that make up the application group | No | |
| Spec | [`ApplicationSpec`](#applicationspec) | Application spec with repo, chart and helm release options | Yes | |

### ApplicationSpec

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| Chart    | [`*ChartRef`](#chartref) | Git or Helm repository info for pulling helm charts | Yes |  |
| Release | [`*Release`](#release) | Helm release info including `HelmRelease` options and helm overlay values | Yes |  |
| Subcharts | [ ][`DAG`](#dag) | Subchart dependencies within an Application | No |  |

### ChartRef

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| Url | `string` | Helm repository repo url | Yes | |
| Name | `string` | Helm chart name | Yes | |
| Version | `string` | Helm chart version | No | `latest` |
| AuthSecretRef | [`*corev1.ObjectReference`](https://pkg.go.dev/k8s.io/api/core/v1#ObjectReference) | Secret containing the Auth credentials for a private helm registry | No | `nil` |

### Release

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| Interval | `metav1.Duration` | Interval at which to reconcile the helm release | No | 5m |
| TargetNamespace | `string` | TargetNamespace overrides the targeted namespace for the Helm release. | Yes | namespace of the `HelmRelease` resource |
| Timeout | `*int64` | Time to wait for any individual Kubernetes operation (like Jobs for hooks) during installation and upgrade operations. | No | `5m` |
| Values | `*apiextensionsv1.JSON` | The values for this Helm release. Values are expected in a standard JSON map format | No | `nil` |
| Install | [`*ReleaseInstallSpec`](#releaseinstallspec) | Configuration for the Helm install actions | No | `nil` |
| Upgrade | [`*ReleaseUpgradeSpec`](#releaseupgradespec) | Configuration for the Helm upgrade actions | No | `nil` |
| Rollback | [`*ReleaseRollbackSpec`](#releaserollbackspec) | Configuration for the Helm rollback actions | No | `nil` |

### ReleaseInstallSpec

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| DisableWait | `bool` | Disables the waiting for resources to be ready by the helm install | No | `false` |

### ReleaseUpgradeSpec

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| DisableWait | `bool` | Disables the waiting for resources to be ready by the helm upgrade | No | `false` |
| Force | `bool` | Forces resource updates through a replacement strategy on helm upgrade  | No | `false` |

### ReleaseRollbackSpec

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| DisableWait | `bool` | Disables the waiting for resources to be ready by the helm rollback | No | `false` |

### DAG

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| Name | `string` | Name of the application | Yes | |
| Dependencies | [ ]`string` | Dependencies on other Applications in the `ApplicationGroup.Spec.Applications` | No | |

## Status

### ApplicationGroupStatus

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| Applications | [ ][`ApplicationStatus`](#applicationstatus) | Status of application's reconciliation process | No | |
| Update | `bool` | The current phase of theapplication;s reconciliation process | No | |
| ObservedGeneration | `int64` | The last object generation that was captured and completed by the reconciler | No | |
| Conditions | [ ][`metav1.Condition`](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Condition) | Conditions for the Application Group | No | |

### ApplicationStatus

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| Name | `string` | Name of the application | No | |
| *inline* | [`ChartStatus`](#chartstatus) | Current status of the chart and associated helm release | No | |
| Subcharts | `map[string]`[`ChartStatus`](#chartstatus) | Current status of the subcharts and associated helm releases | No | |

### ChartStatus

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| Error | `string` | Error message encountered during reconciliation (if any) | No | |
| Version | `string` | Version of the chart | No | |
| Staged | `bool` | True denotes that the chart has been pushed to the staging repo (chartmuseum) | No | |
| Conditions | [ ][`metav1.Condition`](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Condition) | Conditions from the HelmChart object | No |