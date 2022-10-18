package edge

import (
	"context"
	"fmt"
	"os"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"edge.jevv.dev/pkg/controllers"

	corev1 "k8s.io/api/core/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

func (r *KServiceReconciler) reconcileKRevision(ctx context.Context, service *servingv1.Service) kindPreProcessorResult {
	if service == nil {
		return kindPreProcessorResult{Result: ctrl.Result{}}
	}

	shouldCreate := false
	shouldUpdate := false
	shouldDelete := false

	revision := &servingv1.Revision{}

	serviceNamespacedName := types.NamespacedName{Name: service.Name, Namespace: service.Namespace}
	revisionNamespacedName := getRevisionNamespacedName(serviceNamespacedName)

	if err := r.Get(ctx, revisionNamespacedName, revision); err != nil {
		if !apierrors.IsNotFound(err) {
			return kindPreProcessorResult{Err: err}
		}

		shouldCreate = true
	}

	if !kServiceHasComputeOffloadAnnotation(service) {
		shouldDelete = !shouldCreate
		shouldCreate = false
	}

	r.Log.V(3).WithName("revision").Info("debug bool", "shouldCreate", shouldCreate, "shouldUpdate", shouldUpdate, "shouldDelete", shouldDelete)
	r.Log.V(3).WithName("revision").Info("debug name", "revisionName", revisionNamespacedName)

	// TODO: check if we should update

	if shouldCreate {
		r.Log.V(3).WithName("revision").Info("Creating edge proxy route.", "name", revisionNamespacedName)

		r.buildRevision(revisionNamespacedName, revision, service)
		// controllerutil.SetControllerReference(service, revision, r.Scheme)

		if err := r.Create(ctx, revision); err != nil {
			if apierrors.IsConflict(err) {
				return kindPreProcessorResult{Result: ctrl.Result{Requeue: true}}
			}

			r.Log.V(3).WithName("revision").Error(err, "couldn't create revision")

			return kindPreProcessorResult{Err: err}
		}
	} else if shouldUpdate {
		r.Log.V(3).WithName("revision").Info("Updating edge proxy route.", "name", revisionNamespacedName)

		r.buildRevision(revisionNamespacedName, revision, service)
		// controllerutil.SetControllerReference(service, revision, r.Scheme)
		controllers.UpdateLastGenerationAnnotation(revision)

		if err := r.Update(ctx, revision); err != nil {
			if apierrors.IsConflict(err) {
				return kindPreProcessorResult{Result: ctrl.Result{Requeue: true}}
			}

			return kindPreProcessorResult{Err: err}
		}
	} else if shouldDelete {
		r.Log.V(3).WithName("revision").Info("Deleting edge proxy route.", "name", revisionNamespacedName)

		if err := r.Delete(ctx, revision); err != nil {
			if apierrors.IsNotFound(err) {
				return kindPreProcessorResult{}
			}

			return kindPreProcessorResult{Err: err}
		}
	}

	return kindPreProcessorResult{}
}

func (r *KServiceReconciler) buildRevision(namespacedName types.NamespacedName, revision *servingv1.Revision, service *servingv1.Service) {
	annotations := service.Annotations

	if annotations == nil {
		annotations = make(map[string]string)
		revision.Annotations = annotations
	}

	labels := revision.Labels

	if labels == nil {
		labels = make(map[string]string)
		revision.Labels = labels
	}

	revision.Name = namespacedName.Name
	revision.Namespace = namespacedName.Namespace

	labels[controllers.ManagedByLabel] = "true"
	labels[controllers.EdgeLocalLabel] = "true"
	labels[controllers.CreatedByLabel] = "knative-edge-computeoffload-controller"
	labels[controllers.ManagedByLabel] = "knative-edge-computeoffload-controller"

	// annotations[controllers.KnativeNoGCAnnotation] = "true"

	// TODO: set concurrency and timeout
	revision.Spec = servingv1.RevisionSpec{
		PodSpec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  fmt.Sprintf("%s-compute-offload-proxy", namespacedName.Name),
					Image: r.ProxyImage,
					Env: []corev1.EnvVar{
						{Name: "REMOTE_URL", Value: annotations[controllers.RemoteUrlAnnotation]},
						{Name: "HTTP_PROXY", Value: os.Getenv("HTTP_PROXY")},
						{Name: "HTTPS_PROXY", Value: os.Getenv("HTTPS_PROXY")},
						{Name: "NO_PROXY", Value: os.Getenv("NO_PROXY")},
					},
				},
			},
		},
	}
}
