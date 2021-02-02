// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package main

import (
	"flag"
	"os"

	"github.com/Azure/Orkestra/pkg/configurer"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/Azure/Orkestra/pkg/workflow"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/controllers"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	helmopv1 "github.com/fluxcd/helm-operator/pkg/apis/helm.fluxcd.io/v1"
	// +kubebuilder:scaffold:imports
)

const (
	stagingRepoNameEnv = "STAGING_REPO_NAME"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = orkestrav1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme

	// Add Argo Workflow scheme to operator
	_ = v1alpha12.AddToScheme(scheme)

	// Add HelmRelease scheme to operator
	_ = helmopv1.AddToScheme(scheme)
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var configPath string
	var stagingRepoName string
	var tempChartStoreTargetDir string
	var cleanup bool

	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&configPath, "config", "", "The path to the controller config file")
	flag.StringVar(&stagingRepoName, "staging-repo-name", "", "The nickname for the helm registry used for staging artifacts (ENV - STAGING_REPO_URL). NOTE: Flag overrides env value")
	flag.StringVar(&tempChartStoreTargetDir, "chart-store-path", "", "The temporary storage path for the downloaded and staged chart artifacts")
	flag.BoolVar(&cleanup, "cleanup", false, "cleanup the pull/downloaded charts from the temporary storage path")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "fdcf4a0d.azure.microsoft.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if stagingRepoName == "" {
		if s := os.Getenv(stagingRepoNameEnv); s != "" {
			stagingRepoName = s
		} else {
			setupLog.Error(err, "staging repo URL must be set")
			os.Exit(1)
		}
	}

	cfg, err := configurer.NewConfigurer(configPath)
	if err != nil {
		setupLog.Error(err, "unable to create new configurer instance", "controller", "config")
		os.Exit(1)
	}

	cfg.Ctrl.Cleanup = cleanup

	rc, err := registry.NewClient(
		ctrl.Log.Logger, cfg.Ctrl.Registries,
		registry.TargetDir(tempChartStoreTargetDir),
	)
	if err != nil {
		setupLog.Error(err, "unable to create new registry client", "controller", "registry-client")
		os.Exit(1)
	}

	if err = (&controllers.ApplicationReconciler{
		Client:          mgr.GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("Application"),
		Scheme:          mgr.GetScheme(),
		Cfg:             cfg.Ctrl,
		RegistryClient:  rc,
		StagingRepoName: stagingRepoName,
		TargetDir:       tempChartStoreTargetDir,
		Recorder:        mgr.GetEventRecorderFor("application-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Application")
		os.Exit(1)
	}

	sCfg, err := cfg.Ctrl.RegistryConfig(stagingRepoName)
	if err != nil {
		setupLog.Error(err, "unable to find staging repo configuration", "controller", "registry-config")
		os.Exit(1)
	}

	if err = (&controllers.ApplicationGroupReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("ApplicationGroup"),
		Scheme:   mgr.GetScheme(),
		Cfg:      cfg.Ctrl,
		Engine:   workflow.Argo(scheme, mgr.GetClient(), sCfg.URL),
		Recorder: mgr.GetEventRecorderFor("appgroup-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ApplicationGroup")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
