package edge

import (
	"context"
	"strings"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"

	"edge.jevv.dev/pkg/controllers"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

//+kubebuilder:rbac:groups=serving.knative.dev,resources=services,verbs=get;list;watch;create;update;patch;delete

type KServiceReconciler struct {
	client.Client

	Log           logr.Logger
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	RemoteCluster cluster.Cluster

	mirror *MirroringReconciler[*servingv1.Service]
}

func (r *KServiceReconciler) kindGenerator() *servingv1.Service {
	return &servingv1.Service{}
}

func (r *KServiceReconciler) kindMerger(src, dst *servingv1.Service) error {
	if src == nil {
		return nil
	}

	if dst == nil {
		*dst = servingv1.Service{}
	}

	dst.ObjectMeta = metav1.ObjectMeta{
		Name:        src.ObjectMeta.Name,
		Namespace:   src.ObjectMeta.Namespace,
		Annotations: src.ObjectMeta.Annotations,
		Labels:      src.ObjectMeta.Labels,
	}

	src.Spec.DeepCopyInto(&dst.Spec)

	annotations := dst.GetAnnotations()

	if annotations == nil {
		annotations = make(map[string]string)
		dst.SetAnnotations(annotations)
	}

	if src.Status.URL != nil {
		url := src.Status.URL.String()

		if !strings.HasSuffix(url, "/") {
			url += "/"
		}

		annotations[controllers.RemoteUrlAnnotation] = url
	}

	return nil
}

func (r *KServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.mirror.Reconcile(ctx, req)
}

func (r *KServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log = mgr.GetLogger().WithName("namespace-controller")
	r.Recorder = mgr.GetEventRecorderFor("namespace-controller")

	r.mirror = &MirroringReconciler[*servingv1.Service]{
		Log:           r.Log.WithName("mirror"),
		Scheme:        r.Scheme,
		Recorder:      r.Recorder,
		RemoteCluster: r.RemoteCluster,
		KindGenerator: r.kindGenerator,
		KindMerger:    r.kindMerger,
	}

	return r.mirror.NewControllerManagedBy(mgr).
		Complete(r)
}
