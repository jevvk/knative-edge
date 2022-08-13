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
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	if !kServiceHasAnnotation(&service) {
		return ctrl.Result{}, nil
	}

	target := getComputeOffloadTrafficTarget(&service)

	if target != nil {
		return ctrl.Result{}, nil
	}

	var percent int64 = 0

	service.Spec.Traffic = append(service.Spec.Traffic, servingv1.TrafficTarget{
		RevisionName: getRevisionNamespacedName(request.NamespacedName).Name,
		Percent:      &percent,
		Tag:          tag,
	})

	controllers.UpdateLastGenerationAnnotation(&service, &service)

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
