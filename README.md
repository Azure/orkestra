# Orkestra

Orkestra is a cloud-native release orchestration platform that allows you to manage the lifecycle and release of a group of Helm Charts (k8s applications) using Kubernetes CRDs. 
It choreographs complicated application releases, with soft and hard dependencies on other applications/charts, with relative ease using DAG-based workflows. 

## Overview

### What is it?
Orkestra allows for deterministic, sequenced deployment of applications (Helm Charts) by leveraging the popular and mature open-source framework like Argo (Workflows). By making use of Weavework's Flux HelmOperator to automate the helm operation, Orkestra provides a robust, idempotent solution for installation, upgrades, and rollbacks.
Additionally, and optionally, it can also sequence the installation of sub-charts (dependencies) of the parent application, using the same DAG-based Argo workflow.

### What problem does it solve?

Complex application require intelligent release orchestration and lifecycle management. The following scenarios present the need for a robust automation platform that oversees the process of releasing a complex set of applications as a single product.

**Continuous Deployment of mission-critical applications** - like 5G core network functions
- Least Disruption and limiting the blast radius: Mission-critical services must be upgraded in isolation to provide the least disruptive upgrades/rollback
- Non-critical and/or orthogonal application upgrades can be done in parallel.

**Applications have both soft and hard dependencies**
- Most legacy enterprise applications are not built for but are merely adapted to Kubernetes.
    - No readiness probes - applications (or deployment) do not proactively check for dependencies like DBs, Servers, etc. before going into an operationally "ready" state.

## Features

- **Built for Kubernetes** - controllers built using the Kubernetes Operators pattern
- **Easy to use** - familiar declarative spec using Custom Resource Definitions
- **Dependency management** - DAG-based workflows for groups of application charts and their sub-charts using Argo Workflow
- **Works with any CD** - use with any existing CD solution out there, including FluxCD and ArgoCD.
- **Built for GitOps** - describe your desired group of applications and manage them from a single version-controlled declarative file.

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