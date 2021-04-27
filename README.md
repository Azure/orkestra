# Orkestra

[![Build Status](https://dev.azure.com/azure/Orkestra/_apis/build/status/Azure.Orkestra?branchName=main)](https://dev.azure.com/azure/Orkestra/_build/latest?definitionId=95&branchName=main)

<p align="center"><img src="./assets/orkestra-core.png" width="750x" /></p>

Orkestra is a cloud-native release orchestration and lifecycle management (LCM) platform for fine-grained orchestration a group of inter-dependent *"Applications"* (and their inter-dependent subcharts). An *"Application"* may be defined as a [Helm](https://helm.sh/) chart or artifact, with or without [subchart](https://helm.sh/docs/helm/helm_dependency) dependencies.

Sophisticated applications require **intelligent** release orchestration and lifecycle management that is not supported by Helm.

Orkestra works by generating a [DAG](https://en.wikipedia.org/wiki/Directed_acyclic_graph) workflow from the `ApplicationGroup` spec. to orchestrate the **deployment** and in-service **upgrades** of multiple applications within a Kubernetes cluster. At a finer-grain, Orkestra can also order the deployment of subcharts within an application chart by generating an embedded DAG workflow.

Orkestra leverages popular and mature open-source frameworks like [Argo](https://argoproj.github.io/argo/) (Workflows), [Flux Helm Operator](https://github.com/fluxcd/helm-operator) and [Chartmuseum](https://chartmuseum.com/)

## Use Cases 💼

### Reliable (continuous) "Deployment" and in-service "Upgrades" of mission-critical applications

📱 *5G Core* [Network Functions](https://www.sdxcentral.com/resources/glossary/network-function/) (NFs)📱 are 5G applications that rely on a rich ecosystem of **infrastructure** and **PaaS** (platform-as-a-service) components to be up and ready before the Network Functions can be successfully started. This establishes a hard dependency between the NFs and the infra/paas applications. Orkestra solves the dependency problem by honoring these dependencies by using a DAG workflow to deploy the infra/paas components prior to starting the NF application. This also allows for the independent lifecycle management of the infra/paas components and groups of NFs.

## Architecture 🏗

To learn more about how Orkestra works see the [architecture](./docs/architecture.md) docs

## Installation 🧰

For getting started you will need,

- A Kubernetes cluster
- `kubectl` - Kubernetes client
- `helm` - Helm client
- `kubebuilder` - [installation](https://book.kubebuilder.io/quick-start.html#installation)
- `controller-gen` - `GO111MODULE=on go get -v -u sigs.k8s.io/controller-tools/cmd/controller-gen@v0.5.0` (this should be run from outside the Orkestra (go module) repo for it to be installed to your `$GOBIN`)
- (_optional_) `argo` - Argo workflow client (follow the instructions to install the binary from [releases](https://github.com/argoproj/argo/releases)

Install the `ApplicationGroup` and custom resource definitions (CRDs)

Install the orkestra controller and supporting services like, Argo Workflow, Flux Helm-operator and Chartmuseum using the provided helm chart

```shell
helm install orkestra chart/orkestra/  --namespace orkestra --create-namespace
```

You should see resources spin up in the _orkestra_ namespace as shown below,

```shell
> kubectl get all -n orkestra

NAME                                               READY   STATUS      RESTARTS   AGE
pod/orkestra-5fdbb5f777-pnbrx                      1/1     Running     1          167m
pod/orkestra-argo-server-597455875f-d7ldz          1/1     Running     0          167m
pod/orkestra-chartmuseum-7f78f54f5-sq6b7           1/1     Running     0          167m
pod/orkestra-helm-operator-6498c6655-4vn2s         1/1     Running     0          167m
pod/orkestra-workflow-controller-688bc7677-k9hxm   1/1     Running     0          167m

NAME                             TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
service/orkestra-argo-server     ClusterIP   10.96.78.145    <none>        2746/TCP   167m
service/orkestra-chartmuseum     ClusterIP   10.96.222.214   <none>        8080/TCP   167m
service/orkestra-helm-operator   ClusterIP   10.96.192.118   <none>        3030/TCP   167m

NAME                                           READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/orkestra                       1/1     1            1           167m
deployment.apps/orkestra-argo-server           1/1     1            1           167m
deployment.apps/orkestra-chartmuseum           1/1     1            1           167m
deployment.apps/orkestra-helm-operator         1/1     1            1           167m
deployment.apps/orkestra-workflow-controller   1/1     1            1           167m

NAME                                                     DESIRED   CURRENT   READY   AGE
replicaset.apps/orkestra-5fdbb5f777                      1         1         1       167m
replicaset.apps/orkestra-argo-server-597455875f          1         1         1       167m
replicaset.apps/orkestra-chartmuseum-7f78f54f5           1         1         1       167m
replicaset.apps/orkestra-helm-operator-6498c6655         1         1         1       167m
replicaset.apps/orkestra-workflow-controller-688bc7677   1         1         1       167m
```

**(_optional_) Argo Workflow Dashboard**

Start the Argo Workflow Dashboard at http://localhost:2476.

```shell
argo server --browser
...
INFO[2021-02-03T01:02:15.840Z] config map                                    name=workflow-controller-configmap
INFO[2021-02-03T01:02:15.851Z] Starting Argo Server                          instanceID= version=v2.12.6
INFO[2021-02-03T01:02:15.851Z] Creating event controller                     operationQueueSize=16 workerCount=4
INFO[2021-02-03T01:02:15.852Z] Argo Server started successfully on http://localhost:2746
INFO[2021-02-03T01:02:15.852Z] Argo UI is available at http://localhost:2746
```

## Features 🌟

- **Dependency management** - DAG-based workflows for groups of application charts and their sub-charts using Argo Workflow
- **Fail fast during in-service upgrades** - limits the blast radius for failures during in-service upgrade of critical components.
- **Failure Remediation** - rollback to last successful spec on encountering failures during in-service upgrades.
- **Built for Kubernetes** - custom controller built using  [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
- **Easy to use** - familiar declarative spec using Kubernetes [Custom Resource Definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- **Works with any Continous Deployment system** - bring your own CD framework to deploy Orkestra Custom Resources. Works with any Kubernetes compatible Continuous Deployment framework like [FluxCD](https://fluxcd.io/) and [ArgoCD](https://argoproj.github.io/argo-cd/).
- **Built for GitOps** - describe your desired set of applications (and dependencies) declaratively and manage them from a version-controlled git repository.

## Developers 👩‍💻

Follow the development [guide](./docs/developers.md) to get started with building and debugging Orkestra

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
