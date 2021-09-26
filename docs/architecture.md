---
layout: default
title: Architecture 
nav_order: 2
---
# Architecture

## How it works

To solve the complex application orchestration problem Orkestra builds a [Directed Acyclic Graph](https://en.wikipedia.org/wiki/Directed_acyclic_graph) using the application, and it's dependencies and submits it to Argo Workflow. The Workflow nodes use [`workflow-executor`](https://argoproj.github.io/argo-workflows/workflow-executors/) nodes to deploy a [`HelmRelease`](https://fluxcd.io/docs/components/helm/api/#helm.toolkit.fluxcd.io/v2beta1.HelmReleaseSpec) object into the cluster. This `HelmRelease` object signals Flux's HelmOperator to perform a "Helm Action" on the referenced chart.

<p align="center"><img src="./assets/orkestra-core.png" width="750x" /></p>

### Sequence

1. Submit an `ApplicationGroup` custom resource object
2. For each "application" in `ApplicationGroup` download the Helm chart from “primary” Helm Registry
3. For each dependency in the Application chart, if subcharts found in `charts/` directory, push the subcharts and the application chart to the ”staging” Helm Registry (Chart-museum).
4. Generate and submit the Argo (DAG) Workflow
5. In parallel,

- (*Executor nodes aka workflow pods will*) submit and probe the status of the deployed `HelmRelease` CR (`.Status.Phase`)
- (*helm-controller will*) watch and deploy Helm charts referred to by each `HelmRelease` CR to the Kubernetes cluster

### Key Pieces

- The ApplicationGroup custom resource type, on which workflow definitions are based. Orkestra basically uses this as its own deployment template, wrapping the Helm releases within.
- Orkestra’s own operator, which interprets the ApplicationGroup input, initiating the actual workflow steps within Argo Workflow.
- Helm charts are referenced in ApplicationGroup documents. Orkestra caches/stages the required Helm charts in a local repository for which it uses ChartMuseum.
- The actual Helm releases as per workflow steps triggered by Argu are executed through a Helm operator that is part of Orkestra.