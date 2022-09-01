package reflector

import (
	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cluster"

	"edge.jevv.dev/pkg/controllers"
	edgecontrolers "edge.jevv.dev/pkg/controllers/edge"
)

type Reflector struct {
	Log logr.Logger

	remoteCluster              cluster.Cluster
	corev1Reconciler           edgecontrolers.CoreV1Reconciler
	knativeServingv1Reconciler edgecontrolers.KnativeServiceV1Reconciler
}

func New(envs []string, scheme *runtime.Scheme) Reflector {
	remoteCluster := NewRemoteClusterOrDie(func(opts *cluster.Options) {
		opts.NewCache = edgecontrolers.EnvScopedCache(envs)
		opts.Scheme = scheme
	})

	corev1Reconciler := edgecontrolers.CoreV1Reconciler{
		RemoteCluster: remoteCluster,
	}

	knativeServingv1Reconciler := edgecontrolers.KnativeServiceV1Reconciler{
		RemoteCluster: remoteCluster,
	}

	return Reflector{
		remoteCluster:              remoteCluster,
		corev1Reconciler:           corev1Reconciler,
		knativeServingv1Reconciler: knativeServingv1Reconciler,
	}
}

func (r *Reflector) GetReconcilers() []controllers.EdgeReconciler {
	corev1Reconcilers := r.corev1Reconciler.GetInnerReconcilers()
	knativeServingv1Reconcilers := r.knativeServingv1Reconciler.GetInnerReconcilers()

	size := len(corev1Reconcilers) + len(knativeServingv1Reconcilers)

	reconcilers := make([]controllers.EdgeReconciler, size)

	var i int = 0
	i += copy(reconcilers[i:], corev1Reconcilers)
	i += copy(reconcilers[i:], knativeServingv1Reconcilers)

	return reconcilers
}

func (r *Reflector) SetupWithManager(mgr ctrl.Manager) error {
	r.Log = mgr.GetLogger().WithName("reflector")

	r.Log.Info("Adding remote cluster to controller manager.")
	if err := mgr.Add(r.remoteCluster); err != nil {
		return err
	}

	r.Log.Info("Setting up corev1 controller.")
	if err := r.corev1Reconciler.SetupWithManager(mgr); err != nil {
		return err
	}

	r.Log.Info("Setting up knativeservingv1 controller.")
	if err := r.knativeServingv1Reconciler.SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}
