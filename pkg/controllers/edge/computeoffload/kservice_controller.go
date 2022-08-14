package computeoffload

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"edge.jevv.dev/pkg/controllers"
)

type KServiceReconciler struct {
	client.Client
	controllers.EdgeReconciler

	Recorder record.EventRecorder
}

func (r *KServiceReconciler) GetName() string {
	return "KnativeEdgeV1/ComputeOffload/KService"
}

func (r *KServiceReconciler) GetHealthz() healthz.Checker {
	return nil
}

func (r *KServiceReconciler) GetHealthzName() string {
	return "healthz-knative-edge-compute-offload-kservice"
}

func (r *KServiceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	var service servingv1.Service

	if err := r.Get(ctx, request.NamespacedName, &service); err != nil {
		// something deleted the service before reconciling it
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	var revision *servingv1.Revision
	revisionNamespacedName := getRevisionNamespacedName(request.NamespacedName)

	if !kServiceHasAnnotation(&service) {
		// if the service doesn't have an annotation, it can mean 2 things
		//   1. the annotation never existed, so the revision shouldn't exist
		//   2. the annotation was removed, so the revision needs to be removed

		if err := r.Get(ctx, revisionNamespacedName, revision); err != nil {
			if !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		} else if err := r.Delete(ctx, revision); err != nil {
			// something else deleted the revision
			if apierrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}

			// requeue on conflict
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// if the target already exists, then exit early
	if target := getComputeOffloadTrafficTarget(&service); target != nil {
		return ctrl.Result{}, nil
	}

	// check if target revision exists, if it does, change the spec
	// this is because we need to change the service if the service
	// is changed in the remote
	if err := r.Get(ctx, revisionNamespacedName, revision); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// we know the revision exists but the revision isn't set in the service,
	// so we add it as a traffic target

	var percent int64 = 0

	service.Spec.Traffic = append(service.Spec.Traffic, servingv1.TrafficTarget{
		RevisionName: getRevisionNamespacedName(request.NamespacedName).Name,
		Percent:      &percent,
		Tag:          tag,
	})

	controllers.UpdateLastGenerationAnnotation(&service, &service)

	// finally, update the service

	if err := r.Update(ctx, &service); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *KServiceReconciler) Setup(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(
			&source.Kind{Type: &servingv1.Service{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		Watches(
			&source.Kind{Type: &servingv1.Revision{}},
			&handler.EnqueueRequestForOwner{},
			builder.WithPredicates(
				predicate.And(
					predicate.GenerationChangedPredicate{},
					predicate.NewPredicateFuncs(isComputeOffloading)),
			),
		).
		Complete(r)
}
