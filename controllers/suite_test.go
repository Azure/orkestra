// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package controllers

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/Azure/Orkestra/pkg/workflow"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sManager ctrl.Manager
var k8sClient client.Client
var testEnv *envtest.Environment
var tempChartStoreTargetDir string
var portForwardStagingRepoURL string = "http://localhost:8080"
var inClusterstagingRepoURL string = "http://orkestra-chartmuseum.orkestra:8080"

func init() {
	tmp := os.TempDir()
	tempChartStoreTargetDir = tmp
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:  []string{filepath.Join("..", "config", "crd", "bases")},
		UseExistingCluster: boolToBoolPtr(true),
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = scheme.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = orkestrav1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = v1alpha12.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = fluxhelmv2beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})

	rc, err := registry.NewClient(
		ctrl.Log,
		registry.TargetDir(tempChartStoreTargetDir),
	)
	Expect(err).NotTo(HaveOccurred())

	// Register the staging helm repository/registry
	err = rc.AddRepo(&registry.Config{
		Name: "staging",
		URL:  portForwardStagingRepoURL,
	})
	Expect(err).NotTo(HaveOccurred())

	Expect(err).NotTo(HaveOccurred())

	err = (&ApplicationGroupReconciler{
		Client:                  k8sManager.GetClient(),
		Log:                     ctrl.Log.WithName("controllers").WithName("ApplicationGroup"),
		Scheme:                  k8sManager.GetScheme(),
		RegistryClient:          rc,
		StagingRepoName:         "staging",
		Engine:                  workflow.Argo(scheme.Scheme, k8sManager.GetClient(), inClusterstagingRepoURL, 10),
		TargetDir:               tempChartStoreTargetDir,
		Recorder:                k8sManager.GetEventRecorderFor("appgroup-controller"),
		DisableRemediation:      true,
		CleanupDownloadedCharts: false,
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
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func boolToBoolPtr(in bool) *bool {
	return &in
}
