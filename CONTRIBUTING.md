# Requirements
If you're interested in contributing to this project, you'll need:
* Go installed - see this [Getting Started](https://golang.org/doc/install) guide for Go.
* Docker installed - see this [Getting Started](https://docs.docker.com/install/) guide for Docker.
* `Kubebuilder` -  see this [Quick Start](https://book.kubebuilder.io/quick-start.html) guide for installation instructions.
* Kubernetes command-line tool `kubectl` 
* Access to a Kubernetes cluster. Some options are:
	* Locally hosted cluster, such as 
		* [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/)
		* [Kind](https://github.com/kubernetes-sigs/kind)
		* Docker for desktop installed localy with RBAC enabled.
	* Azure Kubernetes Service ([AKS](https://azure.microsoft.com/en-au/services/kubernetes-service/))
		* The [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/?view=azure-cli-latest) will be helpful here
		* Retrieve the config for the AKS cluster with `az aks get-credentials --resource-group $RG_NAME --name $Cluster_NAME`
* Setup access to your cluster using a `kubeconfig` file.  See [here](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/) for more details

Here are a few `kubectl` commands you can run to verify your `kubectl` installation and cluster setup
```sh
    kubectl version
    kubectl config get-contexts
    kubectl config current-context
    kubectl cluster-info
    kubectl get pods -n kube-system
```

# Building and Running the operator

## Basics
The scaffolding for the project is generated using `Kubebuilder`. It is a good idea to become familiar with this [project](https://github.com/kubernetes-sigs/kubebuilder). The [quick start](https://book.kubebuilder.io/quick-start.html) guide is also quite useful.

See `Makefile` at the root directory of the project. By default, executing `make` will build the project and produce an executable at `./bin/manager`

For example, to quick start this assumes dependencies have been downloaded and existing CRDs have been installed. See next section
```sh
    git clone https://github.com/Azure/Orkestra.git
    cd Orkestra
    make
    ./bin/manager
```

Other tasks are defined in the `Makefile`. It would be good to familiarise yourself with them.

## Dependencies
The project uses external Go modules that are required to build/run. In addition, to run successfully, any CRDs defined in the project should be regenerated and installed. 

The following steps should illustrate what is required before the project can be run:
1. `go mod tidy` - download the dependencies (this can take a while and there is no progress bar - need to be patient for this one)
2. `make manifests` - regenerates the CRD manifests
3. `make install` -  installs the CRDs into the cluster
4. `make generate` - generate the code

At this point you will be able to build the binary with `go build -o bin/manager main.go`. Alternatively, this step and the required `make generate` step before hand is covered with the default `make`. 

## Running Tests
Running e2e tests require a configure Kubernetes cluster and Azure s connection (through specified environment variables)
```
make test
```