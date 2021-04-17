FROM debian:latest

RUN apt-get update && \
    apt-get upgrade && \
    apt-get install -y sudo && \
    apt-get install -y curl && \
    apt-get install -y git && \
    apt-get install -y make && \
    apt-get install -y wget && \ 
    curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.18.17/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    sudo mv ./kubectl /usr/local/bin/kubectl && \
    echo installed kubectl && \
    wget -c https://golang.org/dl/go1.16.3.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf go1.16.3.linux-amd64.tar.gz && \
    export PATH=$PATH:/usr/local/go/bin && \
    go version && \
    curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 && \
    chmod 700 get_helm.sh && \
    ./get_helm.sh && \ 
    os=$(go env GOOS) && \
    arch=$(go env GOARCH) && \
    curl -L https://go.kubebuilder.io/dl/2.3.1/${os}/${arch} | tar -xz -C /tmp/ && \
    sudo mv /tmp/kubebuilder_2.3.1_${os}_${arch} /usr/local/kubebuilder && \
    export PATH=$PATH:/usr/local/kubebuilder/bin && \
    git clone https://github.com/Azure/Orkestra && \
    cd Orkestra && \
    make setup-kubebuilder
    # kubectl cluster-info
    # kubectl apply -k ./config/crd && \
    # helm install orkestra chart/orkestra/ --namespace orkestra --create-namespace && \
    # kubectl get pods -A && \ 
    # kubectl apply -f examples/simple/bookinfo.yaml