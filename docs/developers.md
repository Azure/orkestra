---
layout: default
title: Developers
nav_order: 4
---
# Guide for contributors and developers

## Install prerequisites

For getting started, you will need:

- **Go installed** - see this [Getting Started](https://golang.org/doc/install) guide for Go.
- **Docker installed** - see this [Getting Started](https://docs.docker.com/install/) guide for Docker.
- **Kubernetes Cluster** *v0.10.0* or higher. Some options are:
  - Locally hosted cluster, such as
    - [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/) (preferred; used by `Makefile` and this guide)
    - [Minikube](https://minikube.sigs.k8s.io/docs/start/)
  - Cloud-based, such as
    - [AKS](https://azure.microsoft.com/en-us/services/kubernetes-service/)
    - [GKE](https://cloud.google.com/kubernetes-engine)
    - [EKS](https://aws.amazon.com/eks/)
- kubectl *v1.18* or higher - see this [Getting started](https://kubernetes.io/docs/tasks/tools/) guide for kubectl.
- helm *v3.5.2* or higher - see this [Getting started](https://helm.sh/docs/intro/install/) guide for helm.
- `kubebuilder` *v2.3.1* or higher - Install using `make setup-kubebuilder`.
- `controller-gen` *v0.5.0* or higher - Install using `make controller-gen`. This is required to generate the ApplicationGroup CRDS.

  > **NOTE**: `controller-gen` versions *< v0.5.0* will generate an incompatible CRD type.

## Build & Run

To solely build the source code, invoke the `make all` target to,

1. Update the CRDs and associated resource on modifying the API types.
2. Build the orkestra controller binary.

To setup a local environment for debugging and/or testing invoke the `make dev` target.

The `dev` target creates a new `KinD` cluster with a local container registry and deploys all the components of Orkestra apart from the orkestra controller deployment.

```shell
make dev

kind create cluster --config .kind-cluster.yaml --name orkestra
Creating cluster "orkestra" ...
 ✓ Ensuring node image (kindest/node:v1.20.2) 🖼
 ✓ Preparing nodes 📦
 ✓ Writing configuration 📜
 ✓ Starting control-plane 🕹️
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
Set kubectl context to "kind-orkestra"
You can now use your cluster with:

kubectl cluster-info --context kind-orkestra

Have a question, bug, or feature request? Let us know! https://kind.sigs.k8s.io/#community 🙂
helm upgrade --install orkestra chart/orkestra --wait --atomic -n orkestra --create-namespace --values "chart/orkestra/values-ci.yaml"
Release "orkestra" does not exist. Installing it now.
manifest_sorter.go:192: info: skipping unknown hook: "crd-install"
manifest_sorter.go:192: info: skipping unknown hook: "crd-install"
manifest_sorter.go:192: info: skipping unknown hook: "crd-install"
manifest_sorter.go:192: info: skipping unknown hook: "crd-install"
NAME: orkestra
LAST DEPLOYED: Mon Jun  7 18:05:28 2021
NAMESPACE: orkestra
STATUS: deployed
REVISION: 1
TEST SUITE: None
NOTES:
Happy Helming with Azure/Orkestra
```

To runs E2E and UTs invoke the `make test` target as follows,

```shell
make test

go test -v ./... -coverprofile coverage.txt -timeout 25m
?   	github.com/Azure/Orkestra	[no test files]
?   	github.com/Azure/Orkestra/api/v1alpha1	[no test files]
=== RUN   TestAPIs
Running Suite: Controller Suite
===============================
Random Seed: 1622919020
Will run 7 of 7 specs

• [SLOW TEST:195.349 seconds]
ApplicationGroup Controller
/home/runner/work/orkestra/orkestra/controllers/appgroup_controller_test.go:23
  ApplicationGroup
  /home/runner/work/orkestra/orkestra/controllers/appgroup_controller_test.go:25
    Should create Bookinfo spec successfully
    /home/runner/work/orkestra/orkestra/controllers/appgroup_controller_test.go:53
... truncated for brevity ...
--- PASS: TestAPIs (985.79s)
PASS
coverage: 67.4% of statements
ok  	github.com/Azure/Orkestra/controllers	985.849s	coverage: 67.4% of statements
?   	github.com/Azure/Orkestra/pkg/meta	[no test files]
?   	github.com/Azure/Orkestra/pkg/registry	[no test files]
?   	github.com/Azure/Orkestra/pkg/utils	[no test files]
=== RUN   Test_subchartValues
=== RUN   Test_subchartValues/withGlobalSuchart
=== RUN   Test_subchartValues/withOnlyGlobal
=== RUN   Test_subchartValues/withOnlySubchart
=== RUN   Test_subchartValues/withNone
--- PASS: Test_subchartValues (0.00s)
    --- PASS: Test_subchartValues/withGlobalSuchart (0.00s)
    --- PASS: Test_subchartValues/withOnlyGlobal (0.00s)
    --- PASS: Test_subchartValues/withOnlySubchart (0.00s)
    --- PASS: Test_subchartValues/withNone (0.00s)
PASS
coverage: 3.3% of statements
ok  	github.com/Azure/Orkestra/pkg/workflow	0.044s	coverage: 3.3% of statements
```

### Debugging using `delve`

- **<ins>Debugging using [Visual Studio Code](https://code.visualstudio.com/) and [delve](https://github.com/go-delve/delve)</ins>**

  - <ins>[Built-in Debugger](https://code.visualstudio.com/docs/languages/go#_debugging)</ins>
    - Required extensions:
      - ["Golang"](https://marketplace.visualstudio.com/items?itemName=golang.go)

    `.vscode/launch.json`

    > set `--disable-remediation` if you do not wish for the controller to automatically rollback or garbage collect the owned resources (pods, jobs, etc.)

    ```json
      {
      "version": "0.2.0",
      "configurations": [
        {
          "name": "Launch Package",
          "type": "go",
          "request": "launch",
          "mode": "auto",
          "program": "${workspaceFolder}",
          "args": [
            "--debug",
            "--log-level", "3", 
            // "--disable-remediation"
          ]
        }
      ]
    }
    ```

## Cleanup

```shell
make clean

./hack/teardown-kind-with-registry.sh
Deleting cluster "orkestra" ...
```

## Opening a Pull Request

- Fork the [repository](https://github.com/Azure/orkestra).
- Check-in all changed files

- 🚨 Update API docs if any of the types are changed

```shell
make api-docs
```

- Create a new PR against the upstream repository and reference the relevant issue(s) in the PR description.

## Supported Workflow Executors

### Helmrelease Workflow Executor Repository

The code for the *default* workflow executor container can be found at [Orkestra Helmrelease Workflow Executor](https://github.com/Azure/helmrelease-workflow-executor).

### Keptn Workflow Executor Repository

The code for the *keptn* workflow executor container can be found at [Orkestra Keptn Workflow Executor](https://github.com/Azure/keptn-workflow-executor).
