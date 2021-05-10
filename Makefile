
# Image URL to use all building/pushing image targets
IMG ?= azureorkestra/orkestra:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

CI_VALUES ?= "chart/orkestra/values-ci.yaml"
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

dev-up: 
	-kind create cluster --config .kind-cluster.yaml --name orkestra

dev-down: dev-stop
	-kind delete cluster --name orkestra 2>&1

dev-run: dev-up
	helm install orkestra chart/orkestra --wait --atomic -n orkestra --create-namespace --values ${CI_VALUES} 

dev-stop: 
	-helm delete orkestra -n orkestra 2>&1

# Run tests
test: lint
	go test -v ./... -coverprofile coverage.txt

# Build manager binary
manager: generate lint 
	go build -o bin/manager main.go

up: 
	kind create cluster --name orkestra

down: stop
	-kind delete cluster --name orkestra

run: up
	helm install orkestra chart/orkestra --wait --atomic -n orkestra --create-namespace

stop:
	-helm delete orkestra -n orkestra

bookinfo-up: 
	kubectl create -f examples/simple/bookinfo.yaml

bookinfo-down: 
	kubectl delete -f examples/simple/bookinfo.yaml

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

lint:
	golangci-lint run .

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# setup kubebuilder
setup-kubebuilder:
	bash scripts/setup-envtest.sh;
	bash scripts/setup-kubebuilder.sh

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.5.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

test-e2e:
	./testing/validation.sh

.PHONY: all manager dev-up dev-down dev-run dev-stop up down run stop bookinfo-up bookinfo-down manifests generate
