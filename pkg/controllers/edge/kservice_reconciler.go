package edge

import (
	"context"
	"fmt"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"

	"edge.jevv.dev/pkg/controllers"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

func (r *KServiceReconciler) reconcileKService(ctx context.Context, service *servingv1.Service) kindPreProcessorResult {
	if service == nil {
		return kindPreProcessorResult{}
	}

	annotations := service.Annotations

	if annotations == nil {
		annotations = make(map[string]string)
		service.Annotations = annotations
	}

	if service.Status.URL != nil {
		url := service.Status.URL.String()

		if !strings.HasSuffix(url, "/") {
			url += "/"
		}

		annotations[controllers.RemoteUrlAnnotation] = url
	}

	revision := &servingv1.Revision{}

	serviceNamespacedName := types.NamespacedName{Name: service.Name, Namespace: service.Namespace}
	revisionNamespacedName := getRevisionNamespacedName(serviceNamespacedName)

	if !kServiceHasComputeOffloadAnnotation(service) {
		// if the service doesn't have an annotation, it can mean 2 things
		//   1. the annotation never existed, so the revision shouldn't exist
		//   2. the annotation was removed, so the revision needs to be removed

		// if err := r.Get(ctx, revisionNamespacedName, revision); err != nil {
		// 	if !apierrors.IsNotFound(err) {
		// 		return ctrl.Result{}, err
		// 	}

		// 	return ctrl.Result{}, nil
		// } else if err := r.Delete(ctx, revision); err != nil {
		// 	// something else deleted the revision
		// 	if apierrors.IsNotFound(err) {
		// 		return ctrl.Result{}, nil
		// 	}

		// 	// requeue on conflict
		// 	if apierrors.IsConflict(err) {
		// 		return ctrl.Result{Requeue: true}, nil
		// 	}

		// 	return ctrl.Result{}, err
		// }

		return kindPreProcessorResult{}
	}

	// ensure latest target is specified

	if service.Spec.Traffic == nil {
		service.Spec.Traffic = make([]servingv1.TrafficTarget, 0)
	}

	if target := getLatestRevisionTarget(service); target == nil {
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
				return kindPreProcessorResult{Result: ctrl.Result{RequeueAfter: 5 * time.Second}}
			}

			return kindPreProcessorResult{}
		}

		// early exit if generation is the same
		if fmt.Sprint(revision.GetGeneration()) == getRevisionGenerationFromTarget(target) {
			return kindPreProcessorResult{}
		}

		// finally, update the tag
		target.Tag = getTargetTagFromRevision(revision)
	} else {
		// if target revision exists, change the spec
		// this is because we need to change the service if the service
		// is changed in the remote

		if err := r.Get(ctx, revisionNamespacedName, revision); err != nil {
			if apierrors.IsNotFound(err) {
				return kindPreProcessorResult{}
			}

			return kindPreProcessorResult{Err: err}
		}

		// we know the revision exists but the revision isn't set in the service,
		// so we add it as a traffic target

		service.Spec.Traffic = append(service.Spec.Traffic, servingv1.TrafficTarget{
			RevisionName: getRevisionNamespacedName(serviceNamespacedName).Name,
			Tag:          getTargetTagFromRevision(revision),
		})
	}

	return kindPreProcessorResult{ShouldUpdate: true}
}
