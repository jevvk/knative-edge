package edge

import (
	"context"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

type NamespaceReconciler struct {
	client.Client

	Log           logr.Logger
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	RemoteCluster cluster.Cluster
	Envs          []string

	mirror *MirroringReconciler[*corev1.Namespace]
}

func (r *NamespaceReconciler) kindGenerator() *corev1.Namespace {
	return &corev1.Namespace{}
}

func (r *NamespaceReconciler) kindMerger(src, dst *corev1.Namespace) error {
	if src == nil {
		return nil
	}

	src = src.DeepCopy()

	if dst == nil {
		*dst = corev1.Namespace{}
	}

	dst.Name = src.Name
	dst.Namespace = src.Namespace
	dst.Annotations = src.Annotations
	dst.Labels = src.Labels

	return nil
}

func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.mirror.Reconcile(ctx, req)
}

func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager, predicates ...predicate.Predicate) error {
	r.mirror = &MirroringReconciler[*corev1.Namespace]{
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
