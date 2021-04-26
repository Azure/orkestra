FROM debian:latest

RUN apt-get update -y && \
    apt-get upgrade -y && \
    apt-get install sudo -y && \
    apt-get install curl -y && \
    apt-get install git -y && \
    apt-get install make -y && \
    apt-get install wget -y && \ 
    apt-get install jq -y && \
    # install kubectl
    curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.18.17/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    sudo mv ./kubectl /usr/local/bin/kubectl && \
    # install golang
    wget -c https://golang.org/dl/go1.16.3.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf go1.16.3.linux-amd64.tar.gz && \
    export PATH=$PATH:/usr/local/go/bin && \
    # install argo
    curl -sLO https://github.com/argoproj/argo/releases/download/v3.0.2/argo-linux-amd64.gz && \
    gunzip argo-linux-amd64.gz && \
    chmod +x argo-linux-amd64 && \
    mv ./argo-linux-amd64 /usr/local/bin/argo && \
    # install helm
    curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 && \
    chmod 700 get_helm.sh && \
    ./get_helm.sh && \ 
    # install kubebuilder
    os=$(go env GOOS) && \
    arch=$(go env GOARCH) && \
    curl -L https://go.kubebuilder.io/dl/2.3.1/${os}/${arch} | tar -xz -C /tmp/ && \
    sudo mv /tmp/kubebuilder_2.3.1_${os}_${arch} /usr/local/kubebuilder && \
    export PATH=$PATH:/usr/local/kubebuilder/bin && \
    git clone https://github.com/Azure/orkestra && \
    cd orkestra && \
    make setup-kubebuilder