# Design and Architecture

## How it works

To solve the complex application orchestration problem Orkestra builds a [Directed Acyclic Graph](https://en.wikipedia.org/wiki/Directed_acyclic_graph) using the application, and it's dependencies and submits it to Argo Workflow. The Workflow nodes use [`workflow-executor`](https://argoproj.github.io/argo/workflow-executors/) nodes to deploy a [`HelmRelease`](https://docs.fluxcd.io/projects/helm-operator/en/stable/references/helmrelease-custom-resource/#helm.fluxcd.io/v1.HelmReleaseSpec) object into the cluster. This `HelmRelease` object signals Flux's HelmOperator to perform a "Helm Action" on the referenced chart.

<p align="center"><img src="../assets/orkestra-core.png" width="750x" /></p>

1. Submit an `ApplicationGroup` custom resource object
2. For each "application" in `ApplicationGroup` download the Helm chart from “primary” Helm Registry
3. For each dependency in the Application chart, if subcharts found in `charts/` directory, push the subcharts and the application chart to the ”staging” Helm Registry (Chart-museum).
4. Generate and submit the Argo (DAG) Workflow
5. In parallel,

- (*Executor nodes aka workflow pods will*) submit and probe the status of the deployed `HelmRelease` CR (`.Status.Phase`)
- (*helm-operator will*) watch and deploy Helm charts referred to by each `HelmRelease` CR to the Kubernetes cluster
