module github.com/Azure/Orkestra

go 1.13

require (
	github.com/chartmuseum/helm-push v0.9.0
	github.com/fluxcd/helm-operator v1.2.0
	github.com/go-logr/logr v0.1.0
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/google/go-cmp v0.5.2
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/spf13/viper v1.7.0
	go.opencensus.io v0.22.5 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9 // indirect
	golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6 // indirect
	golang.org/x/text v0.3.4 // indirect
	google.golang.org/genproto v0.0.0-20201109203340-2640f1f9cdfb // indirect
	google.golang.org/grpc v1.33.2 // indirect
	helm.sh/helm/v3 v3.3.4
	k8s.io/apimachinery v0.17.5
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.4.0
)

// Hack to import helm-operator package
// Start hack

replace sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.2.9

replace (
	github.com/docker/docker => github.com/moby/moby v1.4.2-0.20200203170920-46ec8731fbce
	github.com/fluxcd/flux => github.com/fluxcd/flux v1.19.0
	github.com/fluxcd/flux/pkg/install => github.com/fluxcd/flux/pkg/install v0.0.0-20200402061723-01a239a69319
	github.com/fluxcd/helm-operator/pkg/install => github.com/fluxcd/helm-operator/pkg/install v0.0.0-20200407140510-8d71b0072a3e
)

// github.com/fluxcd/helm-operator/pkg/install lives in this very reprository, so use that
// Transitive requirement from Helm: https://github.com/helm/helm/blob/v3.1.0/go.mod#L44
replace github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d

// Pin Flux to 1.18.0

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

// End hack
