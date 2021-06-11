#!/bin/sh
#
# Adapted from:
# https://kind.sigs.k8s.io/docs/user/local-registry/

set -o errexit

# If you wish to change the cluster name, reg_name or reg_port, make sure to also update
# the following files:
#                     teardown-kind-with-registry.sh
#                     .kind-cluster.yaml
KIND_CLUSTER_OPTS='--config .kind-cluster.yaml --name orkestra'
reg_name='kind-registry'
reg_port='5000'

# Create registry container unless it already exists
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  echo "> Creating kind Registry container..."
  docker run \
    -d --restart=always -p "127.0.0.1:${reg_port}:5000" --name "${reg_name}" \
    registry:2
fi

# Create a cluster with the local registry enabled in containerd
kind create cluster ${KIND_CLUSTER_OPTS}

# connect the registry to the cluster network
# (the network may already be connected)
docker network connect "kind" "${reg_name}" || true

# Document the local registry
# https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${reg_port}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF