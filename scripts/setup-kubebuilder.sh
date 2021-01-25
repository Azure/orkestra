#!/bin/sh

# Copyright (c) Microsoft Corporation.
# Licensed under the MIT License.


os=$(go env GOOS)
arch=$(go env GOARCH)
TESTBIN_DIR=testbin

# download kubebuilder and extract it to tmp
curl -L https://go.kubebuilder.io/dl/2.3.1/${os}/${arch} | tar -xz -C /tmp/
mkdir $TESTBIN_DIR
sudo mv /tmp/kubebuilder_2.3.1_${os}_${arch} /usr/local/kubebuilder
export PATH=$TESTBIN_DIR/kubebuilder/bin