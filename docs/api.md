---
layout: default
title: API Reference 
nav_order: 3
---

# API Reference

## Application

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| *inline* | [`DAG`](#dag) | Dependencies between the Applications that make up the application group | No | |
| Spec | [`ApplicationSpec`](#applicationspec) | Application spec with repo, chart and helm release options | Yes | |

## ApplicationSpec

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| Chart    | [`*ChartRef`](#chartref) | Git or Helm repository info for pulling helm charts | Yes |  |
| Release | [`*Release`](#release) | Helm release info including `HelmRelease` options and helm overlay values | Yes |  |
| Subcharts | [ ][`DAG`](#dag) | Subchart dependencies within an Application | No |  |

## ApplicationGroupSpec

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| Applications | [ ][`Application`](#application) | Applications that make up the application group | Yes | |
| Interval | [`*metav1.Duration`](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration) | Interval specifies the interval between reconciliations of the `ApplicationGroup` | No | **5s, 30s** (*short requeue interval, long requeue on Success*) |

## ApplicationGroupStatus

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| Applications | [ ][`ApplicationStatus`](#applicationstatus) | Status of application's reconciliation process | No | |
| Update | `bool` | The current phase of theapplication;s reconciliation process | No | |
| ObservedGeneration | `int64` | The last object generation that was captured and completed by the reconciler | No | |
| Conditions | [ ][`metav1.Condition`](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Condition) | No | |

## ApplicationStatus

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| Name | `string` | Name of the application | No | |
| *inline* | [`ChartStatus`](#chartstatus) | Current status of the chart and associated helm release | No | |
| Subcharts | `map[string]`[`ChartStatus`](#chartstatus) | Current status of the subcharts and associated helm releases | No | |

## ChartRef

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| ~~*inline*~~ | ~~[`GitChartSource`](https://docs.fluxcd.io/projects/helm-operator/en/stable/references/helmrelease-custom-resource/#helm.fluxcd.io/v1.GitChartSource)~~ | ~~Not supported~~ | ~~No~~ |  |
| *inline* | [`RepoChartSource`](https://docs.fluxcd.io/projects/helm-operator/en/stable/references/helmrelease-custom-resource/#helm.fluxcd.io/v1.RepoChartSource) | Application Helm chart source from a helm registry | Yes |  |
| AuthSecretRef | [`*corev1.ObjectReference`](https://pkg.go.dev/k8s.io/api/core/v1#ObjectReference) | Secret containing the Auth credentials for a private helm registry | No | `nil` |

## ChartStatus

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| Phase | [`HelmReleasePhase`](https://docs.fluxcd.io/projects/helm-operator/en/stable/references/helmrelease-custom-resource/#helm.fluxcd.io/v1.HelmReleasePhase) | The current state of the generated `HelmRelease` resource | No | |
| Error | `string` | Error message encountered during reconciliation (if any) | No | |
| Version | `string` | Version of the chart | No | |
| Staged | `bool` | True denotes that the chart has been pushed to the staging repo (chartmuseum) | No | |
| Conditions | [ ] [`metav1.Condition`](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Condition)

## Release

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| HelmVersion | `string` | Application helm chart version (v2 or v3) | No | v2 |
| ForceUograde | `bool` | Mark the generated `HelmRelease` to `--force` upgrades. This forces the resource updates through delete/recreate if needed. | No | `false` |
| Wait | `*bool` | Mark the generated `HelmRelease` to wait until all Pods, PVCs, Services, and minimum number of Pods of a Deployment, StatefulSet, or ReplicaSet are in a ready state before marking the release as successful. | No | `nil` |
| TargetNamespace | `string` | TargetNamespace overrides the targeted namespace for the Helm release. | Yes | namespace of the `HelmRelease` resource |
| Timeout | `*int64` | Time to wait for any individual Kubernetes operation (like Jobs for hooks) during installation and upgrade operations. | No | `nil` |
| Values | [`HelmValues`](https://docs.fluxcd.io/projects/helm-operator/en/stable/references/helmrelease-custom-resource/#helm.fluxcd.io/v1.HelmValues) | The values for this Helm release. | No | `nil` |

## DAG

| Field    | Type | Description | Required | Defaults |
|:---------|:-----|:------------|:---------|:---------|
| Name | `string` | Name of the application | Yes | |
| Namespace | `string` | Namespace of the application | Yes | |
| Dependencies | [ ]`string` | Dependencies on other Applications in the `ApplicationGroup.Spec.Applications` | No | |
