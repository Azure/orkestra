# Orkestra

[![Build Status](https://dev.azure.com/azure/Orkestra/_apis/build/status/Azure.Orkestra?branchName=main)](https://dev.azure.com/azure/Orkestra/_build/latest?definitionId=95&branchName=main)

Orkestra is a cloud-native release orchestration platform that allows you to manage the lifecycle and orchestrate the release of groups of Kubernetes [Helm](https://helm.sh/) applications through Kubernetes Custom Resource Objects.
Orkestra works by generating dependency driven DAG workflows to orchestrate the deployment and upgrade of multiple applications within a Kubernetes cluster. Additionally, Orkestra can also orchestrate the deployment of multiple microservice ([helm dependencies](https://helm.sh/docs/helm/helm_dependency/) - sub-charts) within a parent Helm chart.

## Overview

### What is it?

Orkestra renders a DAG based workflow for deploying applications to a Kubernetes cluster by leveraging popular and mature open-source frameworks like [Argo](https://argoproj.github.io/argo/) (Workflows), [Flux Helm Operator](https://github.com/fluxcd/helm-operator) and [Chart-museum](https://chartmuseum.com/)
### What problems does it solve?

Complex applications oftentimes require **intelligent** release orchestration and lifecycle management which is not provided by Helm itself. 

Take, for example, **Continuous Deployment of mission-critical applications** - *like 5G core Network Functions or NFs*

- Network Functions are applications that rely on a rich ecosystem of **infrastructure** and **PaaS** (platform-as-a-service) components to be present on the cluster before they can be deployed. This establishes a hard dependency between the applications and the infra/paas applications. 

## Getting Started

For getting started you will need,

- A Kubernetes cluster
- `kubectl` - Kubernetes client
- `helm` - Helm client
- (_optional_) `argo` - Argo workflow client (follow the instructions to install the binary at https://github.com/argoproj/argo/releases)

Install the `ApplicationGroup` and custom resource definitions (CRDs) using `make install`

```console
/home/nitishm/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
kustomize build config/crd | kubectl apply -f -
customresourcedefinition.apiextensions.k8s.io/applicationgroups.orkestra.azure.microsoft.com configured
customresourcedefinition.apiextensions.k8s.io/applications.orkestra.azure.microsoft.com configured
```

Alternatively, you can use the integrated `kustomize` flag directly to install the CRDs using `kubectl` by issuing the following command - `kubectl -k config/bases`

### Using helm

Install the orkestra controller and supporting services like, Argo Workflow, Flux Helm-operator and Chartmuseum using the provided helm chart

```console
helm install orkestra chart/orkestra/  --namespace orkestra --create-namespace  
```

You should see resources spin up in the _orkestra_ namespace as shown below,

```console
NAME                                                          NAMESPACE  AGE
configmap/orkestra-helm-operator-kube-config                  orkestra   4s     
configmap/orkestra-workflow-controller-configmap              orkestra   4s     
endpoints/orkestra-argo-server                                orkestra   4s     
endpoints/orkestra-chartmuseum                                orkestra   4s     
endpoints/orkestra-helm-operator                              orkestra   4s     
pod/orkestra-544949cdf8-htct9                                 orkestra   4s     
pod/orkestra-argo-server-597455875f-qdbhb                     orkestra   4s     
pod/orkestra-chartmuseum-7f78f54f5-vf8lh                      orkestra   4s     
pod/orkestra-helm-operator-cc9b5776b-xmn6d                    orkestra   4s     
pod/orkestra-workflow-controller-688bc7677-5t5tn              orkestra   4s     
secret/argo-token-mt9gw                                       orkestra   4s     
secret/default-token-2jpsq                                    orkestra   9h     
secret/flux-helm-repositories                                 orkestra   4s     
secret/orkestra-chartmuseum                                   orkestra   4s     
secret/orkestra-helm-operator-git-deploy                      orkestra   4s     
secret/orkestra-token-x5gqx                                   orkestra   4s     
secret/sh.helm.release.v1.orkestra.v1                         orkestra   4s     
serviceaccount/argo                                           orkestra   4s     
serviceaccount/default                                        orkestra   9h     
serviceaccount/orkestra                                       orkestra   4s     
service/orkestra-argo-server                                  orkestra   4s     
service/orkestra-chartmuseum                                  orkestra   4s     
service/orkestra-helm-operator                                orkestra   4s     
deployment.apps/orkestra                                      orkestra   4s     
deployment.apps/orkestra-argo-server                          orkestra   4s     
deployment.apps/orkestra-chartmuseum                          orkestra   4s     
deployment.apps/orkestra-helm-operator                        orkestra   4s     
deployment.apps/orkestra-workflow-controller                  orkestra   4s     
replicaset.apps/orkestra-544949cdf8                           orkestra   4s     
replicaset.apps/orkestra-argo-server-597455875f               orkestra   4s     
replicaset.apps/orkestra-chartmuseum-7f78f54f5                orkestra   4s     
replicaset.apps/orkestra-helm-operator-cc9b5776b              orkestra   4s     
replicaset.apps/orkestra-workflow-controller-688bc7677        orkestra   4s     
endpointslice.discovery.k8s.io/orkestra-argo-server-qfk6p     orkestra   4s     
endpointslice.discovery.k8s.io/orkestra-chartmuseum-x8dqq     orkestra   4s     
endpointslice.discovery.k8s.io/orkestra-helm-operator-w88tw   orkestra   4s   
```

**(_optional_) Argo Workflow Dashboard**

The following command should open a browser window to the Argo Workflow Dashboard (the command port-forwards the Argo server to http://localhost:2476).

```console
argo server --browser

INFO[2021-02-03T01:02:15.839Z]                                               authModes="[server]" baseHRef=/ managedNamespace= namespace=orkestra secure=false
WARN[2021-02-03T01:02:15.840Z] You are running in insecure mode. Learn how to enable transport layer security: https://argoproj.github.io/argo/tls/
WARN[2021-02-03T01:02:15.840Z] You are running without client authentication. Learn how to enable client authentication: https://argoproj.github.io/argo/argo-server-auth-mode/
INFO[2021-02-03T01:02:15.840Z] config map                                    name=workflow-controller-configmap
INFO[2021-02-03T01:02:15.840Z] SSO disabled
INFO[2021-02-03T01:02:15.851Z] Starting Argo Server                          instanceID= version=v2.12.6
INFO[2021-02-03T01:02:15.851Z] Creating event controller                     operationQueueSize=16 workerCount=4
INFO[2021-02-03T01:02:15.852Z] Argo Server started successfully on http://localhost:2746
INFO[2021-02-03T01:02:15.852Z] Argo UI is available at http://localhost:2746
```

## How it works

To solve the complex application orchestration problem Orkestra builds a [Directed Acyclic Graph](https://en.wikipedia.org/wiki/Directed_acyclic_graph) using the application, and it's dependencies and submits it to Argo Workflow. The Workflow nodes use [`workflow-executor`](https://argoproj.github.io/argo/workflow-executors/) nodes to deploy a [`HelmRelease`](https://docs.fluxcd.io/projects/helm-operator/en/stable/references/helmrelease-custom-resource/#helm.fluxcd.io/v1.HelmReleaseSpec) object into the cluster. This `HelmRelease` object signals Flux's HelmOperator to perform a "Helm Action" on the referenced chart.

<p align="center"><img src="./assets/orkestra-core.png" width="750x" /></p>

1. Submit `ApplicationGroup` CRs
2. For each application in `ApplicationGroup` download Helm chart from “primary” Helm Registry
3. (*optional) For each dependency in the Application chart, if dependency chart is embedded in `charts/` directory, push to ”staging” Helm Registry (Chart-museum).
4. Generate and submit Argo Workflow DAG
5. (Executor nodes only) Submit and probe deployment state of `HelmRelease` CR.
6. Fetch and deploy Helm charts referred to by each `HelmRelease` CR to the Kubernetes cluster.
   (*optional) Embedded subcharts are fetched from the “staging” registry instead of the “primary/remote” registry.

## Sequence Diagram

See [sequence diagrams](./docs/SEQUENCE.md)

## Features

- **Built for Kubernetes** - custom controller built using the [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) project
- **Easy to use** - familiar declarative spec using Kubernetes [Custom Resource Definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- **Dependency management** - DAG-based workflows for groups of application charts and their sub-charts using Argo Workflow
- **Works with any Continous Deployment system** - bring your own CD to deploy Orkestra Custom Resources. Works with any Kubernetes compatible Continuous Deployment framework like [FluxCD](https://fluxcd.io/) and [ArgoCD](https://argoproj.github.io/argo-cd/).
- **Built for GitOps** - describe your desired set of applications (and dependencies) declaratively and manage them from a version-controlled git repository.

## Examples

Try out the examples in [examples](./examples)

## Roadmap

### Functional

- [ ] Handling of `ApplicationGroup` UPDATE & DELETE reconcilation events : [#64](https://github.com/Azure/Orkestra/issues/64), [#59](https://github.com/Azure/Orkestra/issues/59)

### Features

- [ ] Rollback ApplicationGroup to previous version on failure by re-deploying last-applied workflow. 
- [ ] Support multiple remediation strategies on failure
- [ ] Make the switch from [helm-operator](https://github.com/fluxcd/helm-operator) to [helm-controller](https://github.com/fluxcd/helm-controller)

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
