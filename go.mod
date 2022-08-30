module github.com/Azure/Orkestra

go 1.16

require (
	github.com/argoproj/argo-workflows/v3 v3.1.8
	github.com/chartmuseum/helm-push v0.9.0
	github.com/fluxcd/helm-controller/api v0.11.2
	github.com/fluxcd/pkg/apis/meta v0.10.0
	github.com/fluxcd/source-controller/api v0.10.0
	github.com/go-logr/logr v1.2.2
	github.com/gofrs/flock v0.8.1
	github.com/google/go-cmp v0.5.6
	github.com/heptiolabs/healthcheck v0.0.0-20180807145615-6ff867650f40
	github.com/jinzhu/copier v0.3.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.19.0
	gopkg.in/DATA-DOG/go-sqlmock.v1 v1.3.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	helm.sh/helm/v3 v3.9.4
	k8s.io/api v0.24.2
	k8s.io/apiextensions-apiserver v0.24.2
	k8s.io/apimachinery v0.24.2
	k8s.io/client-go v0.24.2
	sigs.k8s.io/controller-runtime v0.9.5
	sigs.k8s.io/yaml v1.3.0
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v1.4.2-0.20200203170920-46ec8731fbce
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.19.8
	k8s.io/api => k8s.io/api v0.21.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.0
	k8s.io/client-go => k8s.io/client-go v0.21.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.9.5
)
