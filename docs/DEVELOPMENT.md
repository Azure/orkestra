# Guide for contributors and developers
| Files | Description |
|-------|-------------|
| **api/v1alpha1** | `ApplicationGroup` Custom Resource API definitions and structs.
| **azure-pipelines.yml** | CI workflow for Azure Pipelines
| **chart/orkestra** | Helm chart for Orkestra controller, chartmuseum and argo workflow components.
| **config/crd** | Custom Resource Definition (CRDs) registered with Kubernetes API Server
| **controllers/appgroup_controller.go** | Core controller logic for the `Reconcile()` function. See flow diagram below.
| **controllers/appgroup_reconciler.go** | Logic to reconcile the state by generating new workflows to get the application group to the desired state. 
| **controllers/suite_test.go** | Integration test bootstrap using Ginkgo's Behavior Driven Test framework.
| **Dockerfile** | Dockerfile for building and deploying orkestra controller docker image
| **main.go** | Entrypoint (`func main()`) to the controller. Bootstraps the orkestra controller manager and instantiates all supporting components required by the reconciler.
| **pkg/checksum.go** | `ApplicationGroup` spec checksum utilities
| **pkg/configurer** | Configuration loader that parses the registry configuration config.yaml file.
| **pkg/helm.go** | Wrappers for Helm Actions using the official helmv2 package.
| **pkg/helpers.go** | Miscellaneous utility functions
| **pkg/registry** | Helm Registry functions using the office helmv2 package and chartmuseum for pull and push functionality, respectively
| **pkg/workflow** | DAG workflow generation and submission interface, implemented using Argo Workflows
| **Tiltfile** | `Tilt` is a useful utility for development, that watches files for changes and builds & pushes new docker images to a live pod as and when changes occur. See [docs](https://docs.tilt.dev/) to learn more.

## Reconciler Flow

<p align="center"><img src="./assets/../../assets/reconciler-flow.png" width="750x" /></p>

## Building and running

### Manually
1. Build a docker image and push to your own personal docker registry (careful not to override the latest tag)

```terminal
docker build . -t <your-registry>/orkestra:<your-tag>
docker push <your-registry>/orkestra:<your-tag>
```

2. Update the orkestra deployment with your registry/image:tag
 
```terminal
helm upgrade orkestra chart/orkestra -n orkestra --create-namespace --set image.repository=<your-registry> --set image.tag=<your-tag>
```

### Using Tilt
Install the `tilt` binary using instructions provided at [intallation](https://docs.tilt.dev/install.html)

```terminal
tilt up
Tilt started on http://localhost:10350/
v0.19.0, built 2021-03-19

(space) to open the browser
(s) to stream logs (--stream=true)
(t) to open legacy terminal mode (--legacy=true)
(ctrl-c) to exit
```
