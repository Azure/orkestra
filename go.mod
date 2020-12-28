module github.com/Azure/Orkestra

go 1.13

require (
	github.com/fluxcd/helm-operator v1.2.0
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.5.0
)

// Pin Helm Operator to release v1.2.0
replace (
	github.com/fluxcd/helm-operator v1.2.0 => ./vendor/github.com/fluxcd/helm-operator
	// github.com/fluxcd/helm-operator/pkg/install lives in this very reprository, so use that
	github.com/fluxcd/helm-operator/pkg/install => ./vendor/github.com/fluxcd/helm-operator/pkg/install

)

// Transitive requirement from Helm: https://github.com/helm/helm/blob/v3.1.0/go.mod#L44
replace github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d

// Pin Flux to 1.18.0
replace (
	github.com/fluxcd/flux => github.com/fluxcd/flux v1.18.0
	github.com/fluxcd/flux/pkg/install => github.com/fluxcd/flux/pkg/install v0.0.0-20200206191601-8b676b003ab0
)

// Force upgrade because of a transitive downgrade.
// github.com/fluxcd/helm-operator
// +-> github.com/fluxcd/flux@v1.17.2
//     +-> k8s.io/client-go@v11.0.0+incompatible
replace k8s.io/client-go => k8s.io/client-go v0.17.2

// Force upgrade because of a transitive downgrade.
// github.com/fluxcd/flux
// +-> github.com/fluxcd/helm-operator@v1.0.0-rc6
//     +-> helm.sh/helm/v3@v3.1.2
//     +-> helm.sh/helm@v2.16.1
replace (
	helm.sh/helm/v3 => helm.sh/helm/v3 v3.1.2
	k8s.io/helm => k8s.io/helm v2.16.3+incompatible
)

// Force upgrade because of transitive downgrade.
// runc >=1.0.0-RC10 patches CVE-2019-19921.
// runc >=1.0.0-RC7 patches CVE-2019-5736.
// github.com/fluxcd/helm-operator
// +-> helm.sh/helm/v3@v3.1.2
//     +-> github.com/opencontainers/runc@v0.1.1
replace github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc10

// helm-2to3 package replaces these packages in its go.mod
replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible
