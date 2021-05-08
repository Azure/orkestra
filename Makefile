
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

dev-up: dev-cluster dev-deploy
dev-down: stop
	kind delete cluster orkestra

dev-cluster:
	kind create cluster --config .kind-cluster.yaml

dev-deploy:
	helm install orkestra chart/orkestra --wait --atomic -n orkestra --create-namespace --values ${CI_VALUES} 

# Run tests
test:
	go test -v ./... -coverprofile coverage.txt

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run:
	helm install orkestra chart/orkestra --wait --atomic -n orkestra --create-namespace

stop:
	helm delete orkestra -n orkestra

bookinfo:
	kubectl create -f examples/simple/bookinfo.yaml

make clean-bookinfo:
	kubectl delete -f examples/simple/bookinfo.yaml

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

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