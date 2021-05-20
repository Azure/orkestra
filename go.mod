module github.com/Azure/Orkestra

go 1.15

require (
	github.com/argoproj/argo v2.5.2+incompatible
	github.com/chartmuseum/helm-push v0.9.0
	github.com/fluxcd/helm-controller/api v0.9.0
	github.com/fluxcd/source-controller/api v0.12.2
	github.com/go-logr/logr v0.4.0
	github.com/go-openapi/spec v0.19.5 // indirect
	github.com/gofrs/flock v0.8.0
	github.com/google/go-cmp v0.5.2
	github.com/heptiolabs/healthcheck v0.0.0-20180807145615-6ff867650f40
	github.com/jinzhu/copier v0.3.0
	github.com/kr/text v0.2.0 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/onsi/ginkgo v1.16.1
	github.com/onsi/gomega v1.11.0
	go.opencensus.io v0.22.5 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	gopkg.in/DATA-DOG/go-sqlmock.v1 v1.3.0 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v2 v2.4.0
	helm.sh/helm/v3 v3.5.4
	k8s.io/api v0.20.4
	k8s.io/apiextensions-apiserver v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v1.4.2-0.20200203170920-46ec8731fbce
	k8s.io/api => k8s.io/api v0.20.2
	k8s.io/client-go => k8s.io/client-go v0.20.2
)
