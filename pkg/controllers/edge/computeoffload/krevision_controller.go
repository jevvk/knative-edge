package computeoffload

import (
	"context"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"edge.jevv.dev/pkg/controllers"
)

type KRevisionReconciler struct {
	client.Client
	controllers.EdgeReconciler

	Recorder record.EventRecorder
}

func (r *KRevisionReconciler) GetName() string {
	return "KnativeEdgeV1/ComputeOffload/KRevision"
}

func (r *KRevisionReconciler) GetHealthz() healthz.Checker {
	return nil
}

func (r *KRevisionReconciler) GetHealthzName() string {
	return "healthz-knative-edge-compute-offload-krevision"
}

func kServiceHasAnnotation(service *servingv1.Service) bool {
	if service == nil {
		return false
	}

	if annotations := service.GetAnnotations(); annotations != nil {
		value, exists := annotations[controllers.OffloadToRemoteAnnotation]

		return exists && strings.ToLower(value) == "true"
	}

	return false
}

func (r *KRevisionReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	var service servingv1.Service

	if err := r.Get(ctx, request.NamespacedName, &service); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	var shouldCreate = false
	var shouldUpdate = false
	var shouldDelete = false
	var revision *servingv1.Revision

	revisionNamespacedName := getRevisionNamespacedName(request.NamespacedName)

	if err := r.Get(ctx, revisionNamespacedName, revision); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		shouldCreate = true
	}

	if !shouldCreate && !kServiceHasAnnotation(&service) {
		shouldDelete = true
	}

	// TODO: check if we should update

	if shouldCreate {
		revision = r.buildRevision(revisionNamespacedName, &service)

		if err := r.Create(ctx, revision); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{}, err
		}
	} else if shouldUpdate {
		newRevision := r.buildRevision(revisionNamespacedName, &service)

		controllers.UpdateLastGenerationAnnotation(revision, newRevision)

		if err := r.Update(ctx, newRevision); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{}, err
		}
	} else if shouldDelete {
		if err := r.Delete(ctx, revision); err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}

			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *KRevisionReconciler) buildRevision(namespacedName types.NamespacedName, owner *servingv1.Service) *servingv1.Revision {
	return &servingv1.Revision{
		ObjectMeta: metav1.ObjectMeta{
			Name:            namespacedName.Name,
			Namespace:       namespacedName.Namespace,
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(owner)},
			Labels: map[string]string{
				controllers.ManagedLabel:   "true",
				controllers.EdgeLocalLabel: "true",
				controllers.CreatedByLabel: "knative-edge-computeoffload-controller",
				controllers.ManagedByLabel: "knative-edge-computeoffload-controller",
			},
			Annotations: map[string]string{
				controllers.KnativeNoGCAnnotation: "true",
			},
		},
		// TODO: create remote proxy
		// TODO: pass domain mapping from remote
		// TODO: set concurrency and timeout
		// TODO: pass http proxy envs
		Spec: servingv1.RevisionSpec{},
	}
}

func (r *KRevisionReconciler) Setup(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(
			&source.Kind{Type: &servingv1.Service{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		Complete(r)
}
