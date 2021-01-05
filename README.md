# Orkestra

Orkestra is a cloud-native release orchestration platform that allows you to manage the lifecycle and orchestrate the release of groups of Kubernetes [Helm](https://helm.sh/) applications through Kubernetes Custom Resource Objects.
Orkestra works by generating dependency driven DAG workflows to orchestrate the deployment and upgrade of multiple applications within a Kubernetes cluster. Additionally, Orkestra can also orchestrate the deployment of multiple microservice ([helm dependencies](https://helm.sh/docs/helm/helm_dependency/) - sub-charts) within a parent Helm chart.

## Overview

### What is it?

Orkestra renders a DAG based workflow for deploying applications to a Kubernetes cluster by leveraging popular and mature open-source frameworks like [Argo](https://argoproj.github.io/argo/) (Workflows), [Flux Helm Operator](https://github.com/fluxcd/helm-operator) and *optionally* [Harbor](https://goharbor.io/).

### What problems does it solve?

Complex applications oftentimes require **intelligent** release orchestration and lifecycle management which is not provided by Helm itself. 

Take, for example, **Continuous Deployment of mission-critical applications** - *like 5G core Network Functions or NFs*

- Network Functions are applications that rely on a rich ecosystem of **infrastructure** and **PaaS** (platform-as-a-service) components to be present on the cluster before they can be deployed. This establishes a hard dependency between the applications and the infra/paas applications. 

## How it works

To solve the complex application orchestration problem Orkestra builds a [Directed Acyclic Graph](https://en.wikipedia.org/wiki/Directed_acyclic_graph) using the application, and it's dependencies and submits it to Argo Workflow. The Workflow nodes use [`workflow-executor`](https://argoproj.github.io/argo/workflow-executors/) nodes to deploy a [`HelmRelease`](https://docs.fluxcd.io/projects/helm-operator/en/stable/references/helmrelease-custom-resource/#helm.fluxcd.io/v1.HelmReleaseSpec) object into the cluster. This `HelmRelease` object signals Flux's HelmOperator to perform a "Helm Action" on the referenced chart.

<p align="center"><img src="./assets/orkestra-core.png" width="750x" /></p>

1. Submit `Application` and `ApplicationGroup` CRs
2. For each `Application` in `ApplicationGroup` download Helm chart from “primary” Helm Registry
3. (*optional) For each dependency in the `Application` chart, if dependency chart is embedded in `charts/` directory, push to ”staging” Helm Registry (Harbor).
4. Generate and submit Argo Workflow DAG
5. (Executor nodes only) Submit and probe deployment state of `HelmRelease CR.
6. Fetch and deploy Helm charts referred to by each `HelmRelease` CR to the Kubernetes cluster.
   (*optional) Embedded subcharts are fetched from the “staging” registry instead of the “primary/remote” registry.

## Features

- **Built for Kubernetes** - custom controller built using the [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) project
- **Easy to use** - familiar declarative spec using Kubernetes [Custom Resource Definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- **Dependency management** - DAG-based workflows for groups of application charts and their sub-charts using Argo Workflow
- **Works with any Continous Deployment system** - bring your own CD to deploy Orkestra Custom Resources. Works with any Kubernetes compatible Continuous Deployment framework like [FluxCD](https://fluxcd.io/) and [ArgoCD](https://argoproj.github.io/argo-cd/).
- **Built for GitOps** - describe your desired set of applications (and dependencies) declaratively and manage them from a version-controlled git repository.

## Contributing

For instructions about setting up your environment to develop and extend the operator, please see
[contributing.md](https://github.com/Azure/Orkestra/blob/main/CONTRIBUTING.md)

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and do, grant us
the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

### Reporting security issues and security bugs

For instructions on reporting security issues and bugs, please see [security.md](https://github.com/Azure/Orkestra/blob/main/SECURITY.md)
