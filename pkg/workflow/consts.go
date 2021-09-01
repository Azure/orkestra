package workflow

const (
	EntrypointTemplateName = "entry"

	HelmReleaseArg                 = "helmrelease"
	TimeoutArg                     = "timeout"
	ActionArg                      = "action"
	KeptnConfigmapNameArg          = "keptn-configmap-name"
	KeptnConfigmapNamespaceArg     = "keptn-configmap-namespace"
	HelmReleaseExecutorName        = "helmrelease-executor"
	HelmReleaseReverseExecutorName = "helmrelease-reverse-executor"

	ValuesKeyGlobal = "global"
	ChartLabelKey   = "chart"

	DefaultTimeout = "5m"

	ExecutorName     = "executor"
	ExecutorImage    = "azureorkestra/executor"
	ExecutorImageTag = "v0.4.1"

	KeptnExecutor         = "azureorkestra/keptn-executor"
	KeptnExecutorImageTag = "v0.1.0"

	DefaultExecutorName = "default"
	KeptnExecutorName   = "keptn"

	ChartMuseumName = "chartmuseum"

	Project        = "orkestra"
	OwnershipLabel = "owner"
	HeritageLabel  = "heritage"
)
