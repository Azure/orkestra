# Orkestra

[![Go Reference](https://pkg.go.dev/badge/github.com/Azure/Orkestra.svg)](https://pkg.go.dev/github.com/Azure/Orkestra)
[![GitHub Workflow Status](https://img.shields.io/github/workflow/status/azure/orkestra/E2E%20Testing?label=e2e)](https://github.com/Azure/orkestra/actions)
[![Docker Pulls](https://img.shields.io/docker/pulls/azureorkestra/orkestra)](https://hub.docker.com/r/azureorkestra/orkestra)
[![codecov](https://codecov.io/gh/Azure/orkestra/branch/main/graph/badge.svg?token=7zcSfCKZSw)](https://codecov.io/gh/Azure/orkestra)
![GitHub commits since latest release (by SemVer)](https://img.shields.io/github/commits-since/azure/orkestra/latest)
[![GitHub contributors](https://img.shields.io/github/contributors/azure/orkestra)](https://github.com/Azure/orkestra/graphs/contributors)

Orkestra is a cloud-native **Release Orchestration** and **Lifecycle Management (LCM)** platform for a related group of Helm releases and their subcharts.

Orkestra is built atop popular CNCF projects like,

- [Argo Workflows](https://argoproj.github.io/workflows/),
- [Flux Helm Controller](https://github.com/fluxcd/helm-controller),
- [Chartmuseum](https://chartmuseum.com/)
- [Keptn](https://keptn.sh)

<p align="center"><img src="./assets/orkestra-core.png" width="750x" /></p>

## Background and Motivation

While **Helm** can model a dependencies using **subcharts**, this dependency relation at Helm release time is not very sophisticated at all. Moreover Helm does not support a way to specify a dependency relation between a parent chart and a subchart. A Helm release for Helm is really a true atomic unit wherein in the Helm package dependency tree gets flattened by resource type and is not treated as a node in a dependency graph at all.

In the **ideal** world, pods and their replica sets are either perfectly **stateless** and don’t care about release state of other components to come up correctly.

Using **Helm Hooks**, **Kubernetes Jobs** and **Init Containers**, you might end up with a carefully crafted and working Helm release for a specific combination of components and conditions.

To manage a group of Helm releases, be it sequential releases of charts not having a parent/subchart relationship and/or both, orchestration is needed.

## What is Orkestra?

Orkestra is one solution to introduce Helm release orchestration. Orkestra provides this by building on top of **Argo Workflow**, a workflow engine on top of Kubernetes for workflow orchestration, where each step in a workflow is executed by a Pod. As such, Argo Workflow engine is a more powerful, more flexible adaptation of what **Init Containers** and **Kubernetes Jobs** provide without the orchestration.

Argo enables a DAG based dependency graph with defined workflow steps in the graph and conditions to transition through the graph as well as detailed insight into the graph and its state. Helm releases matching transitions in the graph are executed by the FluxCD Helm controller that Orkestra ships.

### How it works

The unit of deployment for Orkestra based Helm releases is based on a workflow definition with a custom resource type that models the relationship between individual Helm releases making up the whole.

The `ApplicationGroup` spec allows to structure an orchestrated set of releases through grouping Helm releases into an `ApplicationGroup`, either through defining a sequence on non-related charts and/or charts with subcharts where subcharts are not merged into a single release but are executed as a release of their own inside a workflow step.

This is a powerful construct, that provides a unified view and definition on the intent and the status of orchestrated releases without complicating any individual component inside any of the release. It is possible to orchestrate a set of unrelated Helm packages without making changes to these Helm packages that would be required when using Helm hooks, Kubernetes jobs and/or init containers.

## Features 🌟

- **Dependency management** - DAG-based workflows for groups of application charts and their sub-charts using Argo Workflow
- **Fail fast during in-service upgrades** - limits the blast radius for failures during in-service upgrade of critical components.
- **Failure Remediation** - rollback to last successful spec on encountering failures during in-service upgrades.
- **Built for Kubernetes** - custom controller built using  [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
- **Easy to use** - familiar declarative spec using Kubernetes [Custom Resource Definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- **Works with any Continous Deployment system** - bring your own CD framework to deploy Orkestra Custom Resources. Works with any Kubernetes compatible Continuous Deployment framework like [FluxCD](https://fluxcd.io/) and [ArgoCD](https://argoproj.github.io/argo-cd/).
- **Built for GitOps** - describe your desired set of applications (and dependencies) declaratively and manage them from a version-controlled git repository.

## Architecture 🏗

To learn more about how Orkestra works see the [architecture](./docs/architecture.md) docs

## Executors 🏃‍♂️

### Helmrelease Executor

The default executor is responsible for deploying the HelmRelease object passed in as an input parameter to the docker container. The HelmRelease is represented by a base64 encoded YAML string. The executor deploys, watches and polls for the status of the deployed HelmRelease until it either succeeds/fails or it times out.

Source code for the HelmRelease executor is available [here](https://github.com/Azure/helmrelease-workflow-executor)

### Keptn Executor (Work in progress)

An evaluation executor responsible to running evaluations against the Helm release using Keptn's SLO/SLI engine.

Source code for the Keptn executor is available [here](https://github.com/Azure/keptn-workflow-executor)

![Orkestra workflow](./assets/orkestra-gif.gif)

## Use Case 💼

### 5G Network Functions 📱

Network functions are not always operated, deployed, and managed in isolation of each other. Network functions implementing parts of a 3GPP release based 5G core often operate in conjunction with other network functions implementing other parts. For example, the deployment of a single Session Management Function might depend on foundational PaaS services being in place.

## Installation 🧰

For getting started you will need,

- A Kubernetes cluster
- [`kubectl`](https://kubernetes.io/docs/tasks/tools/)
- [`helm`](https://helm.sh/docs/intro/install/)
- [`argo`](https://github.com/argoproj/argo/releases)

### Using Helm

```shell
helm upgrade --install orkestra chart/orkestra/ --namespace orkestra --create-namespace
```

### Argo Workflow Dashboard

```shell
argo server
```

, and open the dashboard in a browser at [http://localhost:2476](http://localhost:2476).

## Developers 👩‍💻

Follow the development [guide](./docs/developers.md) to get started with building and debugging Orkestra

## Community 🧑‍🤝‍🧑

Connect with the Azure Orkestra community:

- GitHub [issues](https://github.com/Azure/orkestra/issues) and [pull requests](https://github.com/Azure/orkestra/pulls) in this repo
- Azure Orkestra Slack: Join the Azure Orkestra [Slack](https://join.slack.com/t/azureorkestra/shared_invite/zt-rowzrite-Hm_eaih4GyjjZXWftuoqPQ)

## Contributing 🎁

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
