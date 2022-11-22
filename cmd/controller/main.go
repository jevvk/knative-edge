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
	"os"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	edgev1alpha1 "edge.jevv.dev/pkg/apis/edge/v1alpha1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	//+kubebuilder:scaffold:imports

	"edge.jevv.dev/pkg/controllers/edge"
	"edge.jevv.dev/pkg/controllers/edge/store"
	"edge.jevv.dev/pkg/controllers/edge/workoffload"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(servingv1.AddToScheme(scheme))
	utilruntime.Must(edgev1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var probeAddr string
	var proxyImage string
	var environments string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")

	flag.StringVar(&proxyImage, "proxy-image", "", "The image of the proxy component.")
	flag.StringVar(&environments, "envs", "", "A list of comma separated list of environments. The edge cluster will only listen and propagate to these environments.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctx := ctrl.SetupSignalHandler()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                        scheme,
		MetricsBindAddress:            metricsAddr,
		Port:                          9443,
		HealthProbeBindAddress:        probeAddr,
		LeaderElection:                true,
		LeaderElectionID:              "ad6e1dd9.edge.jevv.dev",
		LeaderElectionReleaseOnCancel: true,
		NewCache:                      edge.ManagedScopedCache,
	})

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	envs := strings.Split(environments, ",")

	cluster := edge.NewRemoteClusterOrDie(func(opts *cluster.Options) {
		opts.NewCache = edge.EnvScopedCache(envs)
		opts.Scheme = scheme
	})

	if err = mgr.Add(cluster); err != nil {
		setupLog.Error(err, "Unable to setup remote cluster.")
		os.Exit(1)
	}

	trafficStore := store.Store{
		Log: mgr.GetLogger().WithName("edge-traffic-store"),
	}

	if err = mgr.Add(&trafficStore); err != nil {
		setupLog.Error(err, "Unable to setup traffic split store.")
		os.Exit(1)
	}

	hasEdgeLabelPredicate := edge.HasEdgeSyncLabelPredicate(envs)

	if err = (&edge.NamespaceReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		Log:           mgr.GetLogger().WithName("namespace-controller"),
		Recorder:      mgr.GetEventRecorderFor("namespace-controller"),
		RemoteCluster: cluster,
		Envs:          envs,
	}).SetupWithManager(mgr, hasEdgeLabelPredicate); err != nil {
		setupLog.Error(err, "Unable to create controller.", "controller", "namespace")
		os.Exit(1)
	}

	if err = (&edge.SecretReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		Log:           mgr.GetLogger().WithName("secret-controller"),
		Recorder:      mgr.GetEventRecorderFor("secret-controller"),
		RemoteCluster: cluster,
		Envs:          envs,
	}).SetupWithManager(mgr, hasEdgeLabelPredicate); err != nil {
		setupLog.Error(err, "Unable to create controller.", "controller", "secret")
		os.Exit(1)
	}

	if err = (&edge.ConfigMapReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		Log:           mgr.GetLogger().WithName("configmap-controller"),
		Recorder:      mgr.GetEventRecorderFor("configmap-controller"),
		RemoteCluster: cluster,
		Envs:          envs,
	}).SetupWithManager(mgr, hasEdgeLabelPredicate); err != nil {
		setupLog.Error(err, "Unable to create controller.", "controller", "configmap")
		os.Exit(1)
	}

	if err = (&edge.KServiceReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		Log:           mgr.GetLogger().WithName("kservice-controller"),
		Recorder:      mgr.GetEventRecorderFor("kservice-controller"),
		RemoteCluster: cluster,
		ProxyImage:    proxyImage,
		Envs:          envs,
		Store:         &trafficStore,
	}).SetupWithManager(mgr, hasEdgeLabelPredicate); err != nil {
		setupLog.Error(err, "Unable to create controller.", "controller", "kservice")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "Unable to set up health check.")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "Unable to set up ready check.")
		os.Exit(1)
	}

	if err := mgr.Add(&workoffload.EdgeWorkOffload{
		Client: mgr.GetClient(),
		Envs:   envs,
		Log:    mgr.GetLogger().WithName("edge-traffic"),
		Store:  &trafficStore,
	}); err != nil {
		setupLog.Error(err, "Unable to set up edge traffic splitter.")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "Encountered fatal error running manager.")
		os.Exit(1)
	}
}
