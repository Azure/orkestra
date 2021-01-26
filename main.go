// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package main

import (
	"flag"
	"os"

	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/Azure/Orkestra/pkg/workflow"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = orkestrav1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
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

	// TODO (nitishm): Read and populate the config object

	rc, err := registry.NewClient()
	if err != nil {
		setupLog.Error(err, "unable to create new registry client")
		os.Exit(1)
	}

	if err = (&controllers.ApplicationReconciler{
		Client:         mgr.GetClient(),
		Log:            ctrl.Log.WithName("controllers").WithName("Application"),
		Scheme:         mgr.GetScheme(),
		RegistryClient: rc,
		Recorder:       mgr.GetEventRecorderFor("application-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Application")
		os.Exit(1)
	}
	if err = (&controllers.ApplicationGroupReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ApplicationGroup"),
		Scheme: mgr.GetScheme(),
		// FIXME: Staging repo URL should come from env or flag
		Engine:   workflow.Argo(scheme, mgr.GetClient(), ""),
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
