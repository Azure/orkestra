---
layout: default
title: Developers
nav_order: 5
---
# Guide for contributors and developers

## Prerequisites

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
- `kubectl` *v1.18* or higher - see this [Getting started](https://kubernetes.io/docs/tasks/tools/) guide for `kubectl`.
- `helm` *v3.5.2* or higher - see this [Getting started](https://helm.sh/docs/intro/install/) guide for `helm`.
- `kubebuilder` *v2.3.1* or higher - Install using `make setup-kubebuilder`.
- `controller-gen` *v0.5.0* or higher - Install using `make controller-gen`. This is required to generate the ApplicationGroup CRDS.
  
  > **NOTE**: `controller-gen` versions *< v0.5.0* will generate an incompatible CRD type.

## Build & Run

🚧 🚨 Run the following `make` targets everytime the types are changed (`api/xxx_types.go`)

```shell
$ make generate
/usr/local/bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
```

```shell
$ make manifests
/usr/local/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
/Users/nitish_malhotra/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=chart/orkestra/crds
```

Build the docker image and push it to (your own) docker registry

```shell
docker build . -t <your-registry>/orkestra:<your-tag>
docker push <your-registry>/orkestra:<your-tag>
```

*Example*

If you're using docker and want to build and push the image to your local docker registry, run the following command.

```shell
docker build . -t localhost:5000/orkestra:dev
```

Update the orkestra deployment with your own `registry/image:tag`

```shell
helm upgrade orkestra chart/orkestra -n orkestra --create-namespace --set image.repository=<your-registry> --set image.tag=<your-tag> [--disable-remediation]
```

*Example*

If you have build and pushed an image to your local docker registery using the above example, then the follwing command will update the orkestra deployment with your own `registery/image:tag`.

```shell
helm upgrade --install orkestra chart/orkestra -n orkestra --create-namespace --set image.repository=<localhost:5000/orkestra> --set image.tag=dev [--disable-remediation]
```

---

## Testing & Debugging

### E2E Testing

#### Cleanup any previous installation and KinD cluster

```shell
make clean

helm delete orkestra -n orkestra 2>&1
release "orkestra" uninstalled
kind delete cluster --name orkestra 2>&1
Deleting cluster "orkestra" ...
```

#### Start KinD cluster instance with a dev/debug installation of Orkestra helm chart

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

#### Run the E2E test suite

Run the ginkgo based test suite

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
------------------------------
•
------------------------------
• [SLOW TEST:92.649 seconds]
ApplicationGroup Controller
/home/runner/work/orkestra/orkestra/controllers/appgroup_controller_test.go:23
  ApplicationGroup
  /home/runner/work/orkestra/orkestra/controllers/appgroup_controller_test.go:25
    should create the bookinfo spec and then update it
    /home/runner/work/orkestra/orkestra/controllers/appgroup_controller_test.go:154
------------------------------
•
------------------------------
• [SLOW TEST:259.173 seconds]
ApplicationGroup Controller
/home/runner/work/orkestra/orkestra/controllers/appgroup_controller_test.go:23
  ApplicationGroup
  /home/runner/work/orkestra/orkestra/controllers/appgroup_controller_test.go:25
    should succeed to upgrade the versions of helm releases to newer versions
    /home/runner/work/orkestra/orkestra/controllers/appgroup_controller_test.go:244
------------------------------
• [SLOW TEST:381.190 seconds]
ApplicationGroup Controller
/home/runner/work/orkestra/orkestra/controllers/appgroup_controller_test.go:23
  ApplicationGroup
  /home/runner/work/orkestra/orkestra/controllers/appgroup_controller_test.go:25
    should succeed to rollback helm chart versions on failure
    /home/runner/work/orkestra/orkestra/controllers/appgroup_controller_test.go:303
------------------------------
• [SLOW TEST:49.508 seconds]
ApplicationGroup Controller
/home/runner/work/orkestra/orkestra/controllers/appgroup_controller_test.go:23
  ApplicationGroup
  /home/runner/work/orkestra/orkestra/controllers/appgroup_controller_test.go:25
    should create the bookinfo spec and then delete it while in progress
    /home/runner/work/orkestra/orkestra/controllers/appgroup_controller_test.go:374
------------------------------


Ran 7 of 7 Specs in 985.786 seconds
SUCCESS! -- 7 Passed | 0 Failed | 0 Pending | 0 Skipped
You're using deprecated Ginkgo functionality:
=============================================
Ginkgo 2.0 is under active development and will introduce (a small number of) breaking changes.
To learn more, view the migration guide at https://github.com/onsi/ginkgo/blob/v2/docs/MIGRATING_TO_V2.md
To comment, chime in at https://github.com/onsi/ginkgo/issues/711

  You are using a custom reporter.  Support for custom reporters will likely be removed in V2.  Most users were using them to generate junit or teamcity reports and this functionality will be merged into the core reporter.  In addition, Ginkgo 2.0 will support emitting a JSON-formatted report that users can then manipulate to generate custom reports.

  If this change will be impactful to you please leave a comment on https://github.com/onsi/ginkgo/issues/711
  Learn more at: https://github.com/onsi/ginkgo/blob/v2/docs/MIGRATING_TO_V2.md#removed-custom-reporters

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

### Guide for creating e2e tests

[`EnvTests`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest) use [Ginkgo](https://github.com/onsi/ginkgo) and [Gomega](https://github.com/onsi/gomega) libraries for testing and assertion.

Reference:

- [Azure Databricks Operator Tests](https://github.com/microsoft/azure-databricks-operator/blob/0f722a710fea06b86ecdccd9455336ca712bf775/controllers/run_controller_test.go)
- [Migrations Operator Tests](https://github.com/coderanger/migrations-operator/blob/main/integration/integration_test.go)
- [Flux Source Controller Tests](https://github.com/fluxcd/source-controller/blob/main/controllers/gitrepository_controller_test.go)
- [Kubebuilder Samples](https://book.kubebuilder.io/cronjob-tutorial/writing-tests.html)

---

### Debugging

- **<ins>On Build/Dev Machine</ins>**

  The process described in E2E Testing can be used for local debugging of the orkestra controller as well. This is preferred over okteto and bridge-to-kubernetes since it is faster.

- **<ins>Using [Tilt](https://docs.tilt.dev/)</ins>**

  Install `tilt` using the official [installation](https://docs.tilt.dev/install.html) instructions

  ```shell
  $ tilt up
  Tilt started on http://localhost:10350/
  v0.19.0, built 2021-03-19

  (space) to open the browser
  (s) to stream logs (--stream=true)
  (t) to open legacy terminal mode (--legacy=true)
  (ctrl-c) to exit
  ```

  Use the provided [`Tiltfile`](../Tiltfile) to start developing

---

- **<ins>Debugging using [Visual Studio Code](https://code.visualstudio.com/) and [delve](https://github.com/go-delve/delve)</ins>**

  - <ins>[Built-in Debugger](https://code.visualstudio.com/docs/languages/go#_debugging)</ins>
    - Required extensions:
      - ["Golang"](https://marketplace.visualstudio.com/items?itemName=golang.go)

    `.vscode/launch.json`

    ```json
    {
      // Use IntelliSense to learn about possible attributes.
      // Hover to view descriptions of existing attributes.
      // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
      "version": "0.2.0",
      "configurations": [
        {
          "name": "Launch Package",
          "type": "go",
          "request": "launch",
          "mode": "auto",
          "program": "${fileDirname}",
          "args": [
            "--debug"
          ]
        }
      ]
    }
    ```

  - <ins>[Bridge to Kubernetes](https://marketplace.visualstudio.com/items?itemName=mindaro.mindaro) </ins>

    - Required extensions:
      - ["Bridge to Kubernetes"](https://marketplace.visualstudio.com/items?itemName=mindaro.mindaro)
      - ["Kubernetes"](https://marketplace.visualstudio.com/items?itemName=ms-kubernetes-tools.vscode-kubernetes-tools)

    Deploy the orkestra controller using the deployment methods shown above - **Manually** or  using **`Tilt`**

    Once the orkestra helm release has been successfully deployed you can start debugging by following the step-by-step tutorial below -

    ![Bridge to Kubernetes tutorial GIF](./assets/bridge-to-kubernetes-tutorial.gif)

  - <ins>[Okteto (Remote - Kubernetes](https://marketplace.visualstudio.com/items?itemName=okteto.remote-kubernetes))</ins>

    - Required extensions:
      - ["Remote - SSH"](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-ssh)
      - ["Remote - Kubernetes"](https://marketplace.visualstudio.com/items?itemName=okteto.remote-kubernetes)

    Install [okteto](https://okteto.com/) using `[CMD + Shift + p]` (`[Ctrl + Shift + p]` on Windows)  > __"Okteto: Install"__

    A default [`okteto.yml`](../okteto.yml) has been provided with this repository.

    [*optional*] Configure okteto (do this if you wish to use your own okteto.yml) using `[CMD + Shift + p]` > __"Okteto: Create Manifest"__

    Start the okteto debugger using `[CMD + Shift + p]` > __"Okteto: Up"__

---

## Structure

| Package | Files | Description | 🚧 Requires `make generate` & `make manifests` |
|:---------|:-------|:-------------|:----|
| **./** | **Dockerfile** | Docker manifest to build and deploy the orkestra controller docker image | No |
| | **main.go** | Entrypoint (`func main()`) to the controller. Bootstraps the orkestra controller manager and instantiates all supporting components needed by the reconciler. | No |
| | **Tiltfile** | `Tilt` is a useful utility for development, that watches files for changes and builds & pushes new docker images to a live pod as and when changes occur. See [docs](https://docs.tilt.dev/) to learn more. | No |
| | **azure-pipelines.yml** | CI workflow manifest for Azure Pipelines | No |
| **api/** | **v1alpha1/** | `ApplicationGroup` Custom Resource API definitions and structs. | Yes |
| **chart/**| **orkestra/** | Helm chart for Orkestra controller, chartmuseum and argo workflow components. | No |
| **config/** | **crd/** | Custom Resource Definition (CRDs) that must be registered with Kubernetes API Server. This is automatically generated by kubebuilder using `make manifest`. | No |
| **controllers/** | **appgroup_controller.go** | Core controller logic for the `Reconcile()` function. See flow diagram below. | No |
| | **appgroup_reconciler.go** | Logic to reconcile the state of the Application group object, by generating new workflows to get to the desired state. | No |
| | **suite_test.go** | Bootstrap function to run integration tests for the controller using Ginkgo's Behavior Driven Test framework. | No |
| | **utils/** | Miscellaneous utility functions | No |
| | **registry/** | Helm Registry functions using the official helmv2 package and chartmuseum for pull and push functionality, respectively | No |
| | **workflow/** | DAG workflow generation and submission interface, implemented using Argo Workflows | No |
| | **meta/** | `ApplicationGroup` transition states | No |

---

## Workflow

<p align="center"><img src="./assets/reconciler-flow.png" width="750x" /></p>

## Default Workflow Executor

The code for the *default* workflow executor container can be found at [Orkestra Workflow Executor](https://github.com/Azure/orkestra-workflow-executor).
