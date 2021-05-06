---
layout: default
title: Developers
nav_order: 5
---
# Guide for contributors and developers

## Prerequisites

**Kubernetes Cluster** ([KinD](https://kind.sigs.k8s.io/)/Minikube/AKS/GKE/EKS/others) v0.10.0 or higher
**kubectl** - v1.18 or higher
**helm** - v3.5.2 or higher
**kubebuilder** - v2.3.1 or higher (Kubebuilder and controller-runtime binaries. Install using `make setup-kubebuilder` )
**controller-gen** - v0.5.0 or higher (can be installed using `make controller-gen`). This is required to generate the ApplicationGroup CRDs.

> **NOTE**: `controller-gen` versions *<v0.5.0* will generate an incompatible CRD type

## Build & Run

🚧 🚨 Run the following `make` targets everytime the types are changed (`api/xxx_types.go`)

```shell
$ make generate
/usr/local/bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
```

```shell
$ make manifests
/usr/local/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
```

```shell
$ cp config/crd/bases/orkestra.azure.microsoft.com_applicationgroups.yaml chart/orkestra/crds/orkestra.azure.microsoft.com_applicationgroups.yaml
```

- Build the docker image and push it to (your own) docker registry

```shell
$ docker build . -t <your-registry>/orkestra:<your-tag>
$ docker push <your-registry>/orkestra:<your-tag>
```

- Update the orkestra deployment with your own `registry/image:tag`

```shell
$ helm upgrade orkestra chart/orkestra -n orkestra --create-namespace --set image.repository=<your-registry> --set image.tag=<your-tag> [--disable-remediation]
```

*Example*

```shell
$ helm upgrade orkestra chart/orkestra -n orkestra --create-namespace --set image.repository=<azureorkestra/orkestra> --set image.tag=my-tag [--disable-remediation]
```

---

## Testing & Debugging

### E2E Testing

- Create a KinD cluster for E2E testing with port mapping configuration, to access the chartmuseum port from the host.

```shell
$ kind create cluster -name orkestra --config .kind-cluster.yaml
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

Not sure what to do next? 😅  Check out https://kind.sigs.k8s.io/docs/user/quick-start/
```

- Install Orkestra helm chart using E2E CI values.yaml

```shell
$ helm install orkestra chart/orkestra --wait --atomic -n orkestra --create-namespace --values chart/orkestra/values-ci.yaml
manifest_sorter.go:192: info: skipping unknown hook: "crd-install"
manifest_sorter.go:192: info: skipping unknown hook: "crd-install"
manifest_sorter.go:192: info: skipping unknown hook: "crd-install"
manifest_sorter.go:192: info: skipping unknown hook: "crd-install"
NAME: orkestra
LAST DEPLOYED: Wed May  5 14:25:42 2021
NAMESPACE: orkestra
STATUS: deployed
REVISION: 1
TEST SUITE: None
NOTES:
Happy Helming with Azure/Orkestra
```

- Verify orkestra-chartmuseum service reachability

```shell
$ curl http://127.0.0.1:8080/index.yaml
apiVersion: v1
entries: {}
generated: "2021-05-05T21:26:26Z"
serverInfo: {}
```

- Run the tests

```shell
$ make test
go test -v ./... -coverprofile coverage.txt
?       github.com/Azure/Orkestra       [no test files]
?       github.com/Azure/Orkestra/api/v1alpha1  [no test files]
=== RUN   TestAPIs
Running Suite: Controller Suite
===============================
Random Seed: 1620250177
Will run 1 of 1 specs

• [SLOW TEST:10.026 seconds]
ApplicationGroup Controller
/Users/nitish_malhotra/github/azure/orkestra/controllers/appgroup_controller_test.go:11
  Submit Bookinfo ApplicationGroup
  /Users/nitish_malhotra/github/azure/orkestra/controllers/appgroup_controller_test.go:20
    Should create successfully
    /Users/nitish_malhotra/github/azure/orkestra/controllers/appgroup_controller_test.go:21
------------------------------


Ran 1 of 1 Specs in 13.364 seconds
SUCCESS! -- 1 Passed | 0 Failed | 0 Pending | 0 Skipped
--- PASS: TestAPIs (13.36s)
PASS
coverage: 45.2% of statements
ok      github.com/Azure/Orkestra/controllers   14.435s coverage: 45.2% of statements
?       github.com/Azure/Orkestra/pkg   [no test files]
?       github.com/Azure/Orkestra/pkg/meta      [no test files]
?       github.com/Azure/Orkestra/pkg/registry  [no test files]
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
coverage: 3.9% of statements
ok      github.com/Azure/Orkestra/pkg/workflow  0.935s  coverage: 3.9% of statements
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

  Substitute the "Run the [delve](https://github.com/go-delve/delve) against the modified codebase using your IDE (*VSCode* is preferred)

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
| **pkg/** | **helm.go** | Wrappers for Helm Actions using the official helmv2 package. | No |
| | **helpers.go** | Miscellaneous utility functions | No |
| | **registry/** | Helm Registry functions using the office helmv2 package and chartmuseum for pull and push functionality, respectively | No |
| | **workflow/** | DAG workflow generation and submission interface, implemented using Argo Workflows | No |
| | **meta/** | `ApplicationGroup` transition states | No |

---

## Workflow

<p align="center"><img src="./assets/reconciler-flow.png" width="750x" /></p>