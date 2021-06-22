package testutils

import (
	"errors"
	"math/rand"
	"os"
	"time"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/controllers"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/Azure/Orkestra/pkg/workflow"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	tempChartStoreTargetDir string
)

func init() {
	tmp := os.TempDir()
	tempChartStoreTargetDir = tmp
}

func GetTestEnv() (k8sClient client.Client, testEnv *envtest.Environment, err error) {
	//
	// TODO: set logger
	//

	rand.Seed(time.Now().UnixNano())

	k8sClient = nil
	testEnv = &envtest.Environment{
		UseExistingCluster: BoolToBoolPtr(true),
	}

	cfg, err := testEnv.Start()
	if err != nil {
		return
	}
	if cfg == nil {
		err = errors.New("error: rest.Config = nil, expected non nil")
		return
	}

	//
	// TODO: If returning error after this, call testEnv.Stop() before return.
	//

	err = scheme.AddToScheme(scheme.Scheme)
	if err != nil {
		return
	}
	err = orkestrav1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return
	}
	err = v1alpha13.AddToScheme(scheme.Scheme)
	if err != nil {
		return
	}
	err = fluxhelmv2beta1.AddToScheme(scheme.Scheme)
	if err != nil {
		return
	}

	//
	// TODO: fix MetricsBindAddress
	//
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "0",
		Port:               9443,
	})
	if err != nil {
		return
	}

	rc, err := registry.NewClient(ctrl.Log, registry.TargetDir(tempChartStoreTargetDir))
	if err != nil {
		return
	}

	// Register the staging helm repository/registry
	err = rc.AddRepo(&registry.Config{
		Name: "staging",
		URL:  portForwardStagingRepoURL,
	})
	if err != nil {
		return
	}

	baseLogger := ctrl.Log.WithName("controllers").WithName("ApplicationGroup")
	workflowClientBuilder := workflow.NewBuilder(k8sManager.GetClient(), baseLogger).WithStagingRepo(inClusterstagingRepoURL).WithParallelism(10).InNamespace("orkestra")

	err = (&controllers.ApplicationGroupReconciler{
		Client:                  k8sManager.GetClient(),
		Log:                     baseLogger,
		Scheme:                  k8sManager.GetScheme(),
		RegistryClient:          rc,
		StagingRepoName:         "staging",
		WorkflowClientBuilder:   workflowClientBuilder,
		TargetDir:               tempChartStoreTargetDir,
		Recorder:                k8sManager.GetEventRecorderFor("appgroup-controller"),
		DisableRemediation:      false,
		CleanupDownloadedCharts: false,
	}).SetupWithManager(k8sManager)
	if err != nil {
		return
	}

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		//
		// TODO: if err == nil, throw error.
		//
	}()

	k8sClient = k8sManager.GetClient()
	if k8sClient == nil {
		err = errors.New("error: client.Client = nil, expected non nil")
		return
	}
	return
}

func CleanTestEnv(testEnv *envtest.Environment) {
	if testEnv != nil {
		_ = testEnv.Stop()
	}
}

//func CleanHelmReleases(ctx context.Context, k8sClient *client.Client)
