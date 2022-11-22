package edge

import (
	"context"
	"time"

	"edge.jevv.dev/pkg/controllers"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

func (r *KServiceReconciler) reconcileKService(ctx context.Context, service *servingv1.Service) (ctrl.Result, error) {
	logger := r.Log.V(controllers.DebugLevel).WithName("service")

	if service == nil {
		return ctrl.Result{}, nil
	}

	configuration := &servingv1.Configuration{}

	serviceNamespacedName := types.NamespacedName{Name: service.Name, Namespace: service.Namespace}
	configurationNamespacedName := getConfigurationNamespacedName(serviceNamespacedName)

	if !kServiceHasComputeOffloadLabel(service) {
		// if the service doesn't have an annotation, it can mean 2 things
		//   1. the annotation never existed, so the configuration shouldn't exist
		//   2. the annotation was removed, so the configuration needs to be removed

		if err := r.Get(ctx, configurationNamespacedName, configuration); err != nil {
			if !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		} else if err := r.Delete(ctx, configuration); err != nil {
			// requeue on conflict
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			// check if something else deleted the configuration
			if !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}

		// delete target if exists
		removeEdgeProxyTarget(service)

		return ctrl.Result{}, nil
	}

	logger.Info("compute offload enabled", "name", configurationNamespacedName)

	// retrieve latest traffic split target
	trafficSplit, trafficSplitExists := r.Store.Get(serviceNamespacedName.String())

	if !trafficSplitExists {
		trafficSplit = 0
	}

	traffic := make([]servingv1.TrafficTarget, 0, 1)

	// ensure latest target is specified
	if target := getLatestRevisionTarget(service); target != nil {
		var percent int64 = 100 - trafficSplit
		target.Percent = &percent

		traffic = append(traffic, *target)
	} else {
		var percent int64 = 100 - trafficSplit
		var latestRevision bool = true

		traffic = append(traffic, servingv1.TrafficTarget{
			LatestRevision: &latestRevision,
			Percent:        &percent,
		})
	}

	// ensure edge proxy target is specified
	if target := getEdgeProxyTarget(service); target != nil {
		// if the target already exists, check if we need to change the tag

		logger.Info("edge proxy target found", "name", configurationNamespacedName)

		if err := r.Get(ctx, configurationNamespacedName, configuration); err != nil {
			// something is not right, target exists, but revision doesn't
			// requeue after some time
			if apierrors.IsNotFound(err) {
				logger.Info("edge proxy revision not found", "name", configurationNamespacedName)
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			}

			return ctrl.Result{}, err
		}

		logger.Info("debug edge proxy revision", "revision", getTargetNameFromConfiguration(configuration))

		target.RevisionName = getTargetNameFromConfiguration(configuration)
		target.Tag = getTargetTagFromConfiguration(configuration)
		target.Percent = &trafficSplit

		traffic = append(traffic, *target)
	} else {
		// if target revision exists, change the spec
		// this is because we need to change the service if the service
		// is changed in the remote
		logger.Info("edge proxy target not found", "name", configurationNamespacedName)

		if err := r.Get(ctx, configurationNamespacedName, configuration); err != nil {
			if apierrors.IsNotFound(err) {
				logger.Info("edge proxy revision not found", "name", configurationNamespacedName)
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			}

			return ctrl.Result{}, err
		}

		// we know the revision exists but the revision isn't set in the service,
		// so we add it as a traffic target

		logger.Info("debug edge proxy revision", "revision", getTargetNameFromConfiguration(configuration))

		revisionName := getTargetNameFromConfiguration(configuration)

		if revisionName == "" {
			// not ready yet, try again later
			logger.Info("edge proxy revision not ready", "name", configurationNamespacedName)
			return ctrl.Result{RequeueAfter: time.Second * 5}, nil
		}

		traffic = append(traffic, servingv1.TrafficTarget{
			RevisionName: revisionName,
			Tag:          getTargetTagFromConfiguration(configuration),
			Percent:      &trafficSplit,
		})
	}

	// finally, update traffic
	service.Spec.Traffic = traffic

	// logger.Info("debug service", "service", service)

	// requeue every 5 minutes to update the traffic split
	return ctrl.Result{RequeueAfter: time.Minute}, nil
	// return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}
