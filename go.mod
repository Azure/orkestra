module github.com/Azure/Orkestra

go 1.13

require (
	github.com/argoproj/argo v2.5.2+incompatible
	github.com/chartmuseum/helm-push v0.9.0
	github.com/fluxcd/helm-operator v1.2.0
	github.com/go-delve/delve v1.5.1 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/google/go-cmp v0.5.2
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	k8s.io/api v0.17.5
	k8s.io/apimachinery v0.17.5
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/spf13/viper v1.7.0
	go.opencensus.io v0.22.5 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9 // indirect
	golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6 // indirect
	golang.org/x/text v0.3.4 // indirect
	google.golang.org/genproto v0.0.0-20201109203340-2640f1f9cdfb // indirect
	google.golang.org/grpc v1.33.2 // indirect
	helm.sh/helm/v3 v3.3.4

replace (
	github.com/docker/docker => github.com/moby/moby v1.4.2-0.20200203170920-46ec8731fbce
	github.com/fluxcd/flux => github.com/fluxcd/flux v1.19.0
	github.com/fluxcd/flux/pkg/install => github.com/fluxcd/flux/pkg/install v0.0.0-20200402061723-01a239a69319
	github.com/fluxcd/helm-operator/pkg/install => github.com/fluxcd/helm-operator/pkg/install v0.0.0-20200407140510-8d71b0072a3e
)

replace k8s.io/client-go => k8s.io/client-go v0.17.2