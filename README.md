# Orkestra

Orkestra is a cloud-native release orchestration platform that allows you to manage the lifecycle and orchestrate the release of a group of Kubernetes applications, packaged as [Helm](https://helm.sh/) Charts using declarative, Kubernetes Custom Resource Objects.
Orkestra works by generating dependency driven DAG workflows to orchestrate the release of multiple applications within a cluster, and optionally multiple microservice types within an application ([helm dependecies](https://helm.sh/docs/helm/helm_dependency/)) in the parent application chart.

## Overview

### What is it?

Orkestra allows for the deterministic and ordered deployment of applications (Helm Charts) by leveraging popular and mature open-source frameworks like [Argo](https://argoproj.github.io/argo/) (Workflows) and [Flux Helm Operator](https://github.com/fluxcd/helm-operator).

### What problem does it solve?

Complex application oftentimes require **intelligent** release orchestration and lifecycle management which is not provided by Helm itself

#### Example

**Continuous Deployment of mission-critical applications** - like 5G core network functions

- Network Function applications rely on a rich ecosystem of **infrastructure** and **PaaS** (platform-as-a-service) components to be running on the cluster before they can be deployed.

- Lifecycle Management of infrastructure applications must be carried out independent of application groups.

## Features

- **Built for Kubernetes** - custom controllers built using the [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) project
- **Easy to use** - familiar declarative spec using Kubernetes [Custom Resource Definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- **Dependency management** - DAG-based workflows for groups of application charts and their sub-charts using Argo Workflow
- **Works with any Continous Deployment system** - use with any existing CD solution out there, including [FluxCD](https://fluxcd.io/) and [ArgoCD](https://argoproj.github.io/argo-cd/).
- **Built for GitOps** - describe your desired set of applications (and dependencies) declaratively and manage them from a version-controlled repository.

## Contributing

For instructions about setting up your environment to develop and extend the operator, please see
[contributing.md](https://github.com/azure/orkestra/blob/main/docs/CONTRIBUTING.md)

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

### Reporting security issues and security bugs

For instructions on reporting security issues and bugs, please see [security.md](https://github.com/azure/orkestra/blob/main/docs/SECURITY.md)
