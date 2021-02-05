# Uncomment when https://github.com/tilt-dev/tilt-extensions/issues/129 is fixed
load('ext://kubebuilder', 'kubebuilder') 
kubebuilder("azure.microsoft.com", "orkestra", "v1beta1", "*") 

local_resource(
    'deploy',
    './helm.sh',
)

compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o manager main,go'

local_resource(
  'azureorkestra/orkestra',
  compile_cmd,
  deps=['main.go'],
  resource_deps = ['deploy'])

docker_build(
  'azureorkestra/orkestra',
  '.',
  dockerfile='Dockerfile')

yaml = helm(
  'chart/orkestra',
  # The release name, equivalent to helm --name
  name='orkestra',
  # The namespace to install in, equivalent to helm --namespace
  namespace='orkestra',
  # The values file to substitute into the chart.
  values=['./chart/orkestra/values.yaml'],
)

k8s_yaml(yaml)

k8s_yaml(['./config/samples/dev-applicationgroup.yaml', './config/samples/kafka-dev-application.yaml', './config/samples/redis-dev-application.yaml'])