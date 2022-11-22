package edge

import (
	"context"
	"fmt"
	"time"

	"edge.jevv.dev/pkg/controllers"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

func (r *KServiceReconciler) reconcileKService(ctx context.Context, service *servingv1.Service) kindPreProcessorResult {
	logger := r.Log.V(controllers.DebugLevel).WithName("service")

	if service == nil {
		return kindPreProcessorResult{}
	}

	configuration := &servingv1.Configuration{}

	serviceNamespacedName := types.NamespacedName{Name: service.Name, Namespace: service.Namespace}
	configurationNamespacedName := getConfigurationNamespacedName(serviceNamespacedName)

	shouldUpdate := false

	if !kServiceHasComputeOffloadLabel(service) {
		// if the service doesn't have an annotation, it can mean 2 things
		//   1. the annotation never existed, so the configuration shouldn't exist
		//   2. the annotation was removed, so the configuration needs to be removed

		if err := r.Get(ctx, configurationNamespacedName, configuration); err != nil {
			if !apierrors.IsNotFound(err) {
				return kindPreProcessorResult{Err: err}
			}
		} else if err := r.Delete(ctx, configuration); err != nil {
			// requeue on conflict
			if apierrors.IsConflict(err) {
				return kindPreProcessorResult{Result: ctrl.Result{Requeue: true}}
			}

			// check if something else deleted the configuration
			if !apierrors.IsNotFound(err) {
				return kindPreProcessorResult{Err: err}
			}
		}

		// delete target if exists
		shouldUpdate := removeEdgeProxyTarget(service)

		return kindPreProcessorResult{ShouldUpdate: shouldUpdate}
	}

	logger.Info("compute offload enabled", "name", configurationNamespacedName)

	// retrieve latest traffic split target
	trafficSplit, trafficSplitExists := r.Store.Get(serviceNamespacedName.String())

	if !trafficSplitExists {
		trafficSplit = 0
	}

	// ensure latest target is specified

	if service.Spec.Traffic == nil {
		service.Spec.Traffic = make([]servingv1.TrafficTarget, 0)
		shouldUpdate = true
	}

	if target := getLatestRevisionTarget(service); target == nil {
		var percent int64 = 100 - trafficSplit
		var latestRevision bool = true

		service.Spec.Traffic = append(service.Spec.Traffic, servingv1.TrafficTarget{
			LatestRevision: &latestRevision,
			Percent:        &percent,
		})

		shouldUpdate = true
	}

	if target := getEdgeProxyTarget(service); target != nil {
		// if the target already exists, check if we need to change the tag

		logger.Info("edge proxy target found", "name", configurationNamespacedName)

		if err := r.Get(ctx, configurationNamespacedName, configuration); err != nil {
			// something is not right, target exists, but revision doesn't
			// requeue after some time
			if apierrors.IsNotFound(err) {
				logger.Info("edge proxy revision not found", "name", configurationNamespacedName)
				return kindPreProcessorResult{Result: ctrl.Result{RequeueAfter: 5 * time.Second}}
			}

			return kindPreProcessorResult{}
		}

		// early exit if generation is the same
		if fmt.Sprint(configuration.GetGeneration()) == getConfigurationGenerationFromTarget(target) {
			return kindPreProcessorResult{}
		}

		// finally, update the target

		targetTag := getTargetTagFromConfiguration(configuration)

		if target.Tag != targetTag {
			target.Tag = targetTag
			shouldUpdate = true
		}

		if target.Percent == nil || *target.Percent != trafficSplit {
			target.Percent = &trafficSplit
			shouldUpdate = true
		}
	} else {
		// if target revision exists, change the spec
		// this is because we need to change the service if the service
		// is changed in the remote
		logger.Info("edge proxy target not found", "name", configurationNamespacedName)

		if err := r.Get(ctx, configurationNamespacedName, configuration); err != nil {
			if apierrors.IsNotFound(err) {
				logger.Info("edge proxy revision not found", "name", configurationNamespacedName)
				return kindPreProcessorResult{Result: ctrl.Result{RequeueAfter: 5 * time.Second}}
			}

			return kindPreProcessorResult{Err: err}
		}

		// we know the revision exists but the revision isn't set in the service,
		// so we add it as a traffic target

		revisionName := getTargetNameFromConfiguration(configuration)

		if revisionName == "" {
			// not ready yet, try again later
			return kindPreProcessorResult{Result: ctrl.Result{RequeueAfter: time.Second}}
		}

		service.Spec.Traffic = append(service.Spec.Traffic, servingv1.TrafficTarget{
			RevisionName: revisionName,
			Tag:          getTargetTagFromConfiguration(configuration),
			Percent:      &trafficSplit,
		})

		shouldUpdate = true
	}

	// logger.Info("debug service", "service", service)
	logger.Info("reconciler end", "name", serviceNamespacedName, "shouldUpdate", shouldUpdate)

	// requeue every 5 minutes to update the traffic split
	return kindPreProcessorResult{ShouldUpdate: shouldUpdate, Result: ctrl.Result{RequeueAfter: time.Minute}}
	// return kindPreProcessorResult{ShouldUpdate: shouldUpdate, Result: ctrl.Result{RequeueAfter: time.Minute * 5}}
}
