const { events, Job } = require('brigadier')

events.on("exec", (brigadeEvent, project) => {
  console.log("Running on exec")
  let test = new Job("test-runner")
  test.timeout = 1500000
  test.image = "ubuntu"
  test.shell = "bash"

  test.tasks = [
    "apt-get update -y",
    "apt-get upgrade -y",
    "apt-get install curl -y",
    "apt-get install sudo -y",
    "apt-get install git -y",
    "apt-get install make -y",
    "apt-get install wget -y",
    "apt-get install jq -y",
    "curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.18.17/bin/linux/amd64/kubectl",
    "chmod +x ./kubectl",
    "sudo mv ./kubectl /usr/local/bin/kubectl",
    "echo installed kubectl",
    "curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3",
    "chmod 700 get_helm.sh",
    "./get_helm.sh",
    "echo installed helm",
    "wget -c https://golang.org/dl/go1.16.3.linux-amd64.tar.gz",
    "tar -C /usr/local -xzf go1.16.3.linux-amd64.tar.gz",
    "export PATH=$PATH:/usr/local/go/bin",
    "go version",
    "curl -sLO https://github.com/argoproj/argo/releases/download/v3.0.2/argo-linux-amd64.gz",
    "gunzip argo-linux-amd64.gz",
    "chmod +x argo-linux-amd64",
    "mv ./argo-linux-amd64 /usr/local/bin/argo",
    "argo version",
    "git clone https://github.com/Azure/orkestra",
    "echo cloned orkestra",
    "cd orkestra",
    "git checkout remotes/origin/danaya/addtesting",
    "kubectl apply -k ./config/crd",
    "helm install --wait orkestra chart/orkestra/ --namespace orkestra --create-namespace",
    "kubectl apply -f examples/simple/bookinfo.yaml",
    "sleep 30",
    "argo wait bookinfo -n orkestra",
    "make test-e2e"
  ]

  test.run()
})