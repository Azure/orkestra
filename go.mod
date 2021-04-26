module github.com/Azure/Orkestra

go 1.15

require (
	github.com/argoproj/argo v2.5.2+incompatible
	github.com/chartmuseum/helm-push v0.9.0
	github.com/fluxcd/helm-controller/api v0.9.0
	github.com/fluxcd/helm-operator v1.2.0
	github.com/go-logr/logr v0.3.0
	github.com/go-openapi/spec v0.19.5 // indirect
	github.com/gofrs/flock v0.8.0
	github.com/google/go-cmp v0.5.2
	github.com/jinzhu/copier v0.3.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/onsi/ginkgo v1.14.2 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v2 v2.4.0
	helm.sh/helm/v3 v3.3.4
	k8s.io/api v0.20.4
	k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/kubectl v0.20.4 // indirect
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/yaml v1.2.0
)

// Hack to import helm-operator package
// Start hack

replace sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.2.9

replace (
	github.com/docker/docker => github.com/moby/moby v1.4.2-0.20200203170920-46ec8731fbce
	github.com/fluxcd/flux => github.com/fluxcd/flux v1.19.0
	github.com/fluxcd/flux/pkg/install => github.com/fluxcd/flux/pkg/install v0.0.0-20200402061723-01a239a69319
	github.com/fluxcd/helm-operator/pkg/install => github.com/fluxcd/helm-operator/pkg/install v0.0.0-20200407140510-8d71b0072a3e
	k8s.io/api => k8s.io/api v0.20.2
	k8s.io/client-go => k8s.io/client-go v0.20.2
)

// Force upgrade because of a transitive downgrade.
// github.com/fluxcd/flux
// +-> github.com/fluxcd/helm-operator@v1.0.0-rc6
//     +-> helm.sh/helm/v3@v3.1.2
//     +-> helm.sh/helm@v2.16.1
replace (
	helm.sh/helm/v3 => helm.sh/helm/v3 v3.5.3
	k8s.io/helm => k8s.io/helm v2.16.3+incompatible
)

// Force upgrade because of transitive downgrade.
// runc >=1.0.0-RC10 patches CVE-2019-19921.
// runc >=1.0.0-RC7 patches CVE-2019-5736.
// github.com/fluxcd/helm-operator
// +-> helm.sh/helm/v3@v3.1.2
//     +-> github.com/opencontainers/runc@v0.1.1
replace github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc10

// End hack

replace github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
