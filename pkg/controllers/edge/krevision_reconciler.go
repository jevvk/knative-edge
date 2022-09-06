package edge

import (
	"context"
	"fmt"
	"os"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"edge.jevv.dev/pkg/controllers"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

func (r *KServiceReconciler) reconcileKRevision(ctx context.Context, service *servingv1.Service) (ctrl.Result, error) {
	if service == nil {
		return ctrl.Result{}, nil
	}

	shouldCreate := false
	shouldUpdate := false
	shouldDelete := false

	var revision *servingv1.Revision

	serviceNamespacedName := types.NamespacedName{Name: service.Name, Namespace: service.Namespace}
	revisionNamespacedName := getRevisionNamespacedName(serviceNamespacedName)

	if err := r.Get(ctx, revisionNamespacedName, revision); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		shouldCreate = true
	}

	if !shouldCreate && !kServiceHasAnnotation(service) {
		shouldDelete = true
	}

	// TODO: check if we should update

	if shouldCreate {
		revision = r.buildRevision(revisionNamespacedName, service)
		controllerutil.SetControllerReference(service, revision, r.Scheme)

		if err := r.Create(ctx, revision); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{}, err
		}
	} else if shouldUpdate {
		newRevision := r.buildRevision(revisionNamespacedName, service)
		controllerutil.SetControllerReference(service, revision, r.Scheme)
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

func (r *KServiceReconciler) buildRevision(namespacedName types.NamespacedName, service *servingv1.Service) *servingv1.Revision {
	return &servingv1.Revision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
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
