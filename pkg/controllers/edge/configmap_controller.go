package edge

import (
	"context"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

type ConfigMapReconciler struct {
	client.Client

	Log           logr.Logger
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	RemoteCluster cluster.Cluster

	mirror *MirroringReconciler[*corev1.ConfigMap]
}

func (r *ConfigMapReconciler) kindGenerator() *corev1.ConfigMap {
	return &corev1.ConfigMap{}
}

func (r *ConfigMapReconciler) kindMerger(src, dst *corev1.ConfigMap) error {
	if src == nil {
		return nil
	}

	if dst == nil {
		*dst = corev1.ConfigMap{}
	}

	dst.ObjectMeta = metav1.ObjectMeta{
		Name:        src.ObjectMeta.Name,
		Namespace:   src.ObjectMeta.Namespace,
		Annotations: src.ObjectMeta.Annotations,
		Labels:      src.ObjectMeta.Labels,
	}

	dst.Data = src.Data

	return nil
}

func (r *ConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.mirror.Reconcile(ctx, req)
}

func (r *ConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.mirror = &MirroringReconciler[*corev1.ConfigMap]{
		Log:           r.Log.WithName("mirror"),
		Client:        r.Client,
		Scheme:        r.Scheme,
		Recorder:      r.Recorder,
		RemoteCluster: r.RemoteCluster,
		KindGenerator: r.kindGenerator,
		KindMerger:    r.kindMerger,
	}

	return r.mirror.NewControllerManagedBy(mgr).
		Complete(r)
}
