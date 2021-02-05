load('ext://kubebuilder', 'kubebuilder') 
kubebuilder("azure.microsoft.com", "orkestra", "v1beta1", "*") 

load('ext://namespace', 'namespace_yaml')
k8s_yaml(namespace_yaml("orkestra"),allow_duplicates=True)

docker_build(
  'azureorkestra/orkestra',
  '.',
  dockerfile='Dockerfile')

yaml = helm(
  'chart/orkestra',
  name='orkestra',
  namespace='orkestra',
  values=['./chart/orkestra/values.yaml'],
)

k8s_yaml(yaml,allow_duplicates=True) 

k8s_yaml(['./config/samples/dev-applicationgroup.yaml', './config/samples/kafka-dev-application.yaml', './config/samples/redis-dev-application.yaml'],allow_duplicates=True)