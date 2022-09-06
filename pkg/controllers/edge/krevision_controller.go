package edge

import (
	"context"
	"fmt"
	"os"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/kmeta"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"edge.jevv.dev/pkg/controllers"
)

//+kubebuilder:rbac:groups=serving.knative.dev,resources=revisions,verbs=get;list;watch;create;update;patch;delete

type KRevisionReconciler struct {
	client.Client
	controllers.EdgeReconciler

	ProxyImage string
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
		// TODO: pass domain mapping from remote
		// TODO: set concurrency and timeout
		Spec: servingv1.RevisionSpec{
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  fmt.Sprintf("%s-compute-offload-proxy", namespacedName.Name),
						Image: r.ProxyImage,
						Env: []corev1.EnvVar{
							{Name: "HTTP_PROXY", Value: os.Getenv("HTTP_PROXY")},
							{Name: "HTTPS_PROXY", Value: os.Getenv("HTTPS_PROXY")},
							{Name: "NO_PROXY", Value: os.Getenv("NO_PROXY")},
						},
					},
				},
			},
		},
	}
}

func (r *KRevisionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&servingv1.Service{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		Complete(r)
}
