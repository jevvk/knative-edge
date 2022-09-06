package edge

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

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

func (r *KServiceReconciler) computeOffloadReconcile(ctx context.Context, service *servingv1.Service) error {
	if service == nil {
		return nil
	}

	var revision *servingv1.Revision

	serviceNamespacedName := types.NamespacedName{Name: service.Name, Namespace: service.Namespace}
	revisionNamespacedName := getRevisionNamespacedName(serviceNamespacedName)

	if !kServiceHasAnnotation(service) {
		// if the service doesn't have an annotation, it can mean 2 things
		//   1. the annotation never existed, so the revision shouldn't exist
		//   2. the annotation was removed, so the revision needs to be removed

		if err := r.Get(ctx, revisionNamespacedName, revision); err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}

			return nil
		} else if err := r.Delete(ctx, revision); err != nil {
			// something else deleted the revision
			if apierrors.IsNotFound(err) {
				return nil
			}

			// requeue on conflict
			if apierrors.IsConflict(err) {
				// FIXME: requeue
				return nil
			}

			return err
		}

		return nil
	}

	// ensure latest target is specified

	if service.Spec.Traffic == nil {
		service.Spec.Traffic = make([]servingv1.TrafficTarget, 0)
	}

	if target := getLatestResivionTarget(service); target == nil {
		var percent int64 = 100
		var latestRevision bool = true

		service.Spec.Traffic = append(service.Spec.Traffic, servingv1.TrafficTarget{
			LatestRevision: &latestRevision,
			Percent:        &percent,
		})
	}

	if target := getComputeOffloadTarget(service); target != nil {
		// if the target already exists, check if we need to change the tag

		if err := r.Get(ctx, revisionNamespacedName, revision); err != nil {
			// something is not right, target exists, but revision doesn't
			// requeue after some time
			if apierrors.IsNotFound(err) {
				// FIXME: requeue
				return nil
			}

			return nil
		}

		// early exit if generation is the same
		if fmt.Sprint(revision.GetGeneration()) == getRevisionGenerationFromTarget(target) {
			return nil
		}

		// finally, update the tag
		target.Tag = getTargetTagFromRevision(revision)
	} else {
		// if target revision exists, change the spec
		// this is because we need to change the service if the service
		// is changed in the remote

		if err := r.Get(ctx, revisionNamespacedName, revision); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}

			return err
		}

		// we know the revision exists but the revision isn't set in the service,
		// so we add it as a traffic target

		var percent int64 = 0

		service.Spec.Traffic = append(service.Spec.Traffic, servingv1.TrafficTarget{
			RevisionName: getRevisionNamespacedName(serviceNamespacedName).Name,
			Percent:      &percent,
			Tag:          getTargetTagFromRevision(revision),
		})
	}

	// finally, update the service
	controllers.UpdateLastGenerationAnnotation(service, service)

	return nil
}

func (r *KServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.mirror.Reconcile(ctx, req)
}

func (r *KServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.mirror = &MirroringReconciler[*servingv1.Service]{
		Log:              r.Log.WithName("mirror"),
		Scheme:           r.Scheme,
		Recorder:         r.Recorder,
		RemoteCluster:    r.RemoteCluster,
		KindGenerator:    r.kindGenerator,
		KindMerger:       r.kindMerger,
		KindPreProcessor: r.computeOffloadReconcile,
	}

	return r.mirror.NewControllerManagedBy(mgr).
		Owns(
			&servingv1.Revision{},
			builder.WithPredicates(
				predicate.And(
					predicate.GenerationChangedPredicate{},
					predicate.NewPredicateFuncs(isComputeOffloadRevision)),
			),
		).
		Complete(r)
}
