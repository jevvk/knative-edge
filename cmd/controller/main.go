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

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	edgev1 "edge.jevv.dev/pkg/apis/edge/v1"
	"edge.jevv.dev/pkg/controllers"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	//+kubebuilder:scaffold:imports

	edgecontrollers "edge.jevv.dev/pkg/controllers/edge"
	"edge.jevv.dev/pkg/controllers/edge/computeoffload"
	"edge.jevv.dev/pkg/reflector"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(servingv1.AddToScheme(scheme))
	utilruntime.Must(edgev1.AddToScheme(scheme))
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
		NewCache:                      edgecontrollers.ManagedScopedCache,
	})

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	reflector := reflector.New(strings.Split(environments, ","))
	offloader := computeoffload.New(proxyImage)

	reflectorReconcilers := reflector.GetReconcilers()
	offloaderReconcilers := offloader.GetReconcilers()

	reconcilers := make([]controllers.EdgeReconciler, len(reflectorReconcilers)+len(offloaderReconcilers))
	i := copy(reconcilers[:], reflectorReconcilers)
	i += copy(reconcilers[i:], offloaderReconcilers)

	for _, reconciler := range reconcilers {
		if err = reconciler.Setup(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", reconciler.GetName())
			os.Exit(1)
		}
	}

	for _, reconciler := range reconcilers {
		checker := reconciler.GetHealthz()

		if checker == nil {
			continue
		}

		if err := mgr.AddHealthzCheck(reconciler.GetHealthzName(), checker); err != nil {
			setupLog.Error(err, fmt.Sprintf("unable to set up health check for controller %s", reconciler.GetName()))
			os.Exit(1)
		}
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
