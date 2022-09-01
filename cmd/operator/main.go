/*
Copyright 2022 jevv k.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
	ctrlcfgv1alpha1 "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"

	"edge.jevv.dev/pkg/controllers"

	edgev1 "edge.jevv.dev/pkg/apis/edge/v1"
	operatorv1 "edge.jevv.dev/pkg/apis/operator/v1"
	operatorcontrollers "edge.jevv.dev/pkg/controllers/operator"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(edgev1.AddToScheme(scheme))
	utilruntime.Must(operatorv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

type defaultManagerOptions struct {
	runtime.Object
}

func (o defaultManagerOptions) Complete() (ctrlcfgv1alpha1.ControllerManagerConfigurationSpec, error) {
	leaderElect := true

	return ctrlcfgv1alpha1.ControllerManagerConfigurationSpec{
		Metrics: ctrlcfgv1alpha1.ControllerMetrics{
			BindAddress: ":8080",
		},
		Health: ctrlcfgv1alpha1.ControllerHealth{
			HealthProbeBindAddress: ":8081",
		},
		LeaderElection: &configv1alpha1.LeaderElectionConfiguration{
			LeaderElect:  &leaderElect,
			ResourceName: "d0dsk0s.operator.edge.jevv.dev",
		},
	}, nil
}

func main() {
	var configFile string
	var metricsAddr string
	var probeAddr string

	var proxyImage string
	var controllerImage string

	defaultSyncPeriod := 1 * time.Minute
	remoteSyncPeriod := 5 * time.Minute

	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")

	flag.StringVar(&metricsAddr, "metrics-bind-address", "", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", "", "The address the probe endpoint binds to.")

	flag.StringVar(&proxyImage, "proxy-image", "", "The image of the proxy component.")
	flag.StringVar(&controllerImage, "controller-image", "", "The image of the controller component.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctx := ctrl.SetupSignalHandler()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	var err error
	ctrlConfig := operatorv1.OperatorConfig{}

	options := ctrl.Options{Scheme: scheme}
	// override flags
	options.MetricsBindAddress = metricsAddr
	options.HealthProbeBindAddress = probeAddr
	options.LeaderElectionReleaseOnCancel = true

	// set defaults (shouldn't be overwritten)
	options, _ = options.AndFrom(defaultManagerOptions{})

	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(&ctrlConfig))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}

	if ctrlConfig.Options.RemoteSyncPeriod != nil {
		remoteSyncPeriod = ctrlConfig.Options.RemoteSyncPeriod.Duration
	}

	if ctrlConfig.Options.Namespaces == nil {
		options.Namespace = ""
		setupLog.Info("Operator namespaces are not set. All namespaces will be watched.")
	} else {
		namespaces := withSystemNamespace(*ctrlConfig.Options.Namespaces)
		options.NewCache = cache.MultiNamespacedCacheBuilder(namespaces)
		options.Namespace = ""

		setupLog.Info(fmt.Sprintf("Operator will watch the following namespaces: %s.", strings.Join(*ctrlConfig.Options.Namespaces, ", ")))
	}

	if options.SyncPeriod == nil {
		options.SyncPeriod = &defaultSyncPeriod
	}

	setupLog.Info(fmt.Sprintf("Local cluster cache sync period set to %s.", *options.SyncPeriod))
	setupLog.Info(fmt.Sprintf("Remote cluster cache sync period set to %s.", remoteSyncPeriod))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&operatorcontrollers.EdgeReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		Recorder:         mgr.GetEventRecorderFor("operator-knativeedge"),
		ProxyImage:       proxyImage,
		ControllerImage:  controllerImage,
		RemoteSyncPeriod: remoteSyncPeriod,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Edge")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func withSystemNamespace(namespaces []string) []string {
	exists := false

	for _, namespace := range namespaces {
		if namespace == controllers.SystemNamespace {
			exists = true
			break
		}
	}

	if exists {
		return namespaces
	}

	return append(namespaces, controllers.SystemNamespace)
}
