package edge

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"edge.jevv.dev/pkg/controllers"
	"edge.jevv.dev/pkg/controllers/edge/store"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

//+kubebuilder:rbac:groups=serving.knative.dev,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=serving.knative.dev,resources=revisions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=serving.knative.dev,resources=configurations,verbs=get;list;watch;create;update;patch;delete

type KServiceReconciler struct {
	client.Client

	Log           logr.Logger
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	RemoteCluster cluster.Cluster
	RemoteUrl     string

	ProxyImage string
	Envs       []string
	Store      *store.Store

	mirror *MirroringReconciler[*servingv1.Service]
}

func kServiceHasComputeOffloadLabel(service *servingv1.Service) bool {
	if service == nil {
		return false
	}

	if labels := service.Labels; labels != nil {
		value, exists := labels[controllers.EdgeOffloadLabel]

		return exists && strings.ToLower(value) == "true"
	}

	return false
}

func (r *KServiceReconciler) kindGenerator() *servingv1.Service {
	return &servingv1.Service{}
}

func (r *KServiceReconciler) kindMerger(src, dst *servingv1.Service) error {
	if src == nil {
		return nil
	}

	src = src.DeepCopy()

	if dst == nil {
		*dst = servingv1.Service{}
	}

	dst.Name = src.Name
	dst.Namespace = src.Namespace
	dst.Labels = src.Labels
	dst.Spec.ConfigurationSpec = src.Spec.ConfigurationSpec

	annotations := dst.Annotations

	if annotations == nil {
		annotations = make(map[string]string)
		dst.Annotations = annotations
	}

	if src.Status.URL != nil {
		annotations[controllers.RemoteHostAnnotation] = src.Status.URL.Host
	}

	annotations[controllers.RemoteUrlAnnotation] = r.RemoteUrl

	return nil
}

func (r *KServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.mirror.Reconcile(ctx, req)
}

func (r *KServiceReconciler) SetupWithManager(mgr ctrl.Manager, predicates ...predicate.Predicate) error {
	if r.Store == nil {
		return fmt.Errorf("no traffic split store provided")
	}

	r.mirror = &MirroringReconciler[*servingv1.Service]{
		Log:               r.Log.WithName("mirror"),
		Client:            r.Client,
		Scheme:            r.Scheme,
		Recorder:          r.Recorder,
		RemoteCluster:     r.RemoteCluster,
		Envs:              r.Envs,
		KindGenerator:     r.kindGenerator,
		KindMerger:        r.kindMerger,
		KindPreProcessors: &[]kindPreProcessor[*servingv1.Service]{r.reconcileKConfiguration, r.reconcileKService},
	}

	return r.mirror.NewControllerManagedBy(mgr, predicates...).
		Owns(
			&servingv1.Configuration{},
			builder.WithPredicates(
				predicate.GenerationChangedPredicate{},
				predicate.NewPredicateFuncs(isEdgeProxyConfiguration),
			),
		).
		Complete(r)
}
