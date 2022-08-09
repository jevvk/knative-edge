package reflector

import (
	"sigs.k8s.io/controller-runtime/pkg/cluster"

	"edge.jevv.dev/pkg/controllers"
	edgecontrolers "edge.jevv.dev/pkg/controllers/edge"
)

type Reflector struct {
	corev1Reconciler           edgecontrolers.CoreV1Reconciler
	knativeServingv1Reconciler edgecontrolers.KnativeServiceV1Reconciler
}

func New(envs []string) (*Reflector, error) {
	remoteCluster := NewRemoteClusterOrDie(func(opts *cluster.Options) {
		opts.NewCache = edgecontrolers.EnvScopedCache(envs)
	})

	corev1Reconciler := edgecontrolers.CoreV1Reconciler{
		RemoteCluster: remoteCluster,
	}

	knativeServingv1Reconciler := edgecontrolers.KnativeServiceV1Reconciler{
		RemoteCluster: remoteCluster,
	}

	return &Reflector{
		corev1Reconciler:           corev1Reconciler,
		knativeServingv1Reconciler: knativeServingv1Reconciler,
	}, nil
}

func (r Reflector) GetReconcilers() []controllers.EdgeReconciler {
	corev1Reconcilers := r.corev1Reconciler.GetInnerReconcilers()
	knativeServingv1Reconcilers := r.knativeServingv1Reconciler.GetInnerReconcilers()

	size := len(corev1Reconcilers) + len(knativeServingv1Reconcilers)

	reconcilers := make([]controllers.EdgeReconciler, size)

	var i int = 0
	i += copy(reconcilers[i:], corev1Reconcilers)
	i += copy(reconcilers[i:], knativeServingv1Reconcilers)

	return reconcilers
}
