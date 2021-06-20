#!/bin/sh

set -o errexit

reg_name='kind-registry'

# Check if kind registry container exists.
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  echo "> Kind Registry container ${reg_name} doesn't exist ..."
  echo "> Stopping prepopulating kind registry ..."
  exit 0
fi

all_images=(
    "azureorkestra/orkestra:latest"
    "chartmuseum/chartmuseum:v0.12.0"
    "fluxcd/helm-controller:v0.9.0"
    "fluxcd/source-controller:v0.10.0"
    "quay.io/argoproj/argocli:v3.0.2"
    "quay.io/argoproj/workflow-controller:v3.0.2"
)

# Pull, tag, and push all images to local registry.
for image in ${all_images[@]}; do
    echo ""
    echo "> 🔨  Pull, tag, and push ${image}"
    docker pull ${image}
    docker tag ${image} localhost:5000/${image}
    docker push localhost:5000/${image}
done
