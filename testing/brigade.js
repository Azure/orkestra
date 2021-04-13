const { events, Job } = require('brigadier')
const { KindJob } = require("@brigadecore/brigade-utils");

events.on("exec", (brigadeEvent, project) => {
  let kind = new KindJob("kind");

  kind.tasks.push(
    // add basic tools to image
    "apk update",
    "apk add --update --no-cache git",
    "apk add --update --no-cache sudo",
    "apk add --update --no-cache gcc",
    // install Helm
    "curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3",
    "chmod 700 get_helm.sh",
    "./get_helm.sh",
    // clone Orkestra repo
    "git clone https://github.com/Azure/Orkestra",
    "cd Orkestra",
    "make setup-kubebuilder",
    "kubectl apply -k ./config/crd",
    "helm install orkestra chart/orkestra/  --namespace orkestra --create-namespace",
    // apply example project
    "kubectl apply -f examples/simple/bookinfo.yaml",
    "helm ls -A"
  );

  return kind.run();
})