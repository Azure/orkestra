load('ext://namespace', 'namespace_yaml')
k8s_yaml(namespace_yaml("orkestra"),allow_duplicates=True)

compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o manager main.go'

local_resource(
  'azure-orkestra',
  compile_cmd,
  deps=['main.go'])

docker_build(
  'azureorkestra/orkestra',
  '.',
  dockerfile='Dockerfile')

yaml = local('helm template --namespace orkestra orkestra chart/orkestra --no-hooks --include-crds')

k8s_yaml(yaml,allow_duplicates=True) 

k8s_yaml(['./config/samples/bookinfo.yaml'],allow_duplicates=True)
