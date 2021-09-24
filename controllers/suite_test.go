package controllers_test

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	v1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/controllers"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/Azure/Orkestra/pkg/workflow"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	portForwardStagingRepoURL = "http://127.0.0.1:8080"
	inClusterstagingRepoURL   = "http://orkestra-chartmuseum.orkestra:8080"
)

var (
	k8sClient               client.Client
	tempChartStoreTargetDir string
	testEnv                 *envtest.Environment
)

func init() {
	tmp := os.TempDir()
	tempChartStoreTargetDir = tmp
}

func TestAppGroupController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "ApplicationGroup Controller Suite", []Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	var (
		cfg        *rest.Config
		err        error
		k8sManager ctrl.Manager
	)

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	rand.Seed(time.Now().UnixNano())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		UseExistingCluster: boolToBoolPtr(true),
	}

	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = scheme.AddToScheme(scheme.Scheme)
	Expect(err).ToNot(HaveOccurred())
	err = v1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).ToNot(HaveOccurred())
	err = v1alpha13.AddToScheme(scheme.Scheme)
	Expect(err).ToNot(HaveOccurred())
	err = fluxhelmv2beta1.AddToScheme(scheme.Scheme)
	Expect(err).ToNot(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: fmt.Sprintf(":%d", 8081+config.GinkgoConfig.ParallelNode),
		Port:               9443,
	})
	Expect(err).ToNot(HaveOccurred())

	rc, err := registry.NewClient(ctrl.Log, registry.TargetDir(tempChartStoreTargetDir))
	Expect(err).ToNot(HaveOccurred())

	// Register the staging helm repository/registry
	err = rc.AddRepo(&registry.Config{
		Name: "staging",
		URL:  portForwardStagingRepoURL,
	})
	Expect(err).ToNot(HaveOccurred())

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
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.WorkflowStatusReconciler{
		Client:                k8sManager.GetClient(),
		Log:                   baseLogger,
		Scheme:                k8sManager.GetScheme(),
		WorkflowClientBuilder: workflowClientBuilder,
		Recorder:              k8sManager.GetEventRecorderFor("appgroup-controller"),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	gexec.KillAndWait(5 * time.Second)

	if testEnv != nil {
		err := testEnv.Stop()
		Expect(err).ToNot(HaveOccurred())
	}
})
