
# Image URL to use all building/pushing image targets
IMG ?= azureorkestra/orkestra:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
DEBUG_LEVEL ?= 1
CI_VALUES ?= "chart/orkestra/values-ci.yaml"
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Create a local docker registry, start kinD cluster, and install Orkestra
dev:
	bash hack/kind-with-registry.sh
	helm upgrade --install orkestra chart/orkestra --wait --atomic -n orkestra --create-namespace --values ${CI_VALUES}

debug: dev
	go run main.go --debug --log-level ${DEBUG_LEVEL}

# Delete the Orkestra installation, local docker registry, and the kinD cluster
clean:
	helm delete orkestra -n orkestra 2>&1
	bash hack/teardown-kind-with-registry.sh

ginkgo-test: install
	go get github.com/onsi/ginkgo/ginkgo
	ginkgo ./... -cover -coverprofile coverage.txt

# Run tests
test: install
	go test -v ./... -coverprofile coverage.txt -timeout 25m

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

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
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=chart/orkestra/crds

# Prepare code for PR
prepare-for-pr: vet fmt api-docs

# Generate API reference documentation
api-docs: gen-crd-api-reference-docs
	$(API_REF_GEN) -api-dir=./api/v1alpha1 -config=./hack/api-docs/config.json -template-dir=./hack/api-docs/template -out-file=./docs/api.md

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
	bash hack/setup-envtest.sh;
	bash hack/setup-kubebuilder.sh

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

# Find or download gen-crd-api-reference-docs
gen-crd-api-reference-docs:
ifeq (, $(shell which gen-crd-api-reference-docs))
	@{ \
	set -e ;\
	API_REF_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$API_REF_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get github.com/ahmetb/gen-crd-api-reference-docs@v0.2.0 ;\
	rm -rf $$API_REF_GEN_TMP_DIR ;\
	}
API_REF_GEN=$(GOBIN)/gen-crd-api-reference-docs
else
API_REF_GEN=$(shell which gen-crd-api-reference-docs)
endif
