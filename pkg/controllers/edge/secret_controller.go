package edge

import (
	"context"
	"time"

	"edge.jevv.dev/pkg/controllers"
	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1 "k8s.io/api/core/v1"
)

//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

type SecretReconciler struct {
	client.Client

	Log           logr.Logger
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	RemoteCluster cluster.Cluster
	Envs          []string

	mirror *MirroringReconciler[*corev1.Secret]
}

func (r *SecretReconciler) kindGenerator() *corev1.Secret {
	return &corev1.Secret{}
}

func (r *SecretReconciler) kindMerger(src, dst *corev1.Secret) error {
	if src == nil {
		return nil
	}

	src = src.DeepCopy()

	if dst == nil {
		*dst = corev1.Secret{}
	}

	dst.Name = src.Name
	dst.Namespace = src.Namespace
	dst.Annotations = src.Annotations
	dst.Labels = src.Labels
	dst.Data = src.Data

	return nil
}

func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//////// debug controller time
	start := time.Now()

	defer func() {
		end := time.Now()
		r.Log.V(controllers.DebugLevel).Info("debug reconcile loop", "durationMs", end.Sub(start).Milliseconds())
	}()
	/////// end debug controller time

	return r.mirror.Reconcile(ctx, req)
}

func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager, predicates ...predicate.Predicate) error {
	r.mirror = &MirroringReconciler[*corev1.Secret]{
		Log:           r.Log.WithName("mirror"),
		Client:        r.Client,
		Scheme:        r.Scheme,
		Recorder:      r.Recorder,
		RemoteCluster: r.RemoteCluster,
		Envs:          r.Envs,
		KindGenerator: r.kindGenerator,
		KindMerger:    r.kindMerger,
	}

	return r.mirror.NewControllerManagedBy(mgr, predicates...).
		Complete(r)
}
