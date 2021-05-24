compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o manager main.go'

local_resource(
  'azure-orkestra',
  compile_cmd,
  deps=['main.go'])

docker_build(
  'azureorkestra/orkestra',
  '.',
  dockerfile='Dockerfile')

yaml = local('helm template orkestra chart/orkestra --no-hooks --include-crds')

k8s_yaml(yaml,allow_duplicates=True) 

k8s_yaml(['./config/samples/bookinfo.yaml'],allow_duplicates=True)
