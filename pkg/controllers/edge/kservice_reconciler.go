package edge

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"edge.jevv.dev/pkg/controllers"
	"edge.jevv.dev/pkg/controllers/utils"
	"edge.jevv.dev/pkg/workoffload/strategy"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

func (r *KServiceReconciler) reconcileKService(ctx context.Context, service *servingv1.Service) (ctrl.Result, error) {
	debug := r.Log.WithName("service").V(controllers.DebugLevel)

	if service == nil {
		return ctrl.Result{}, nil
	}

	//////// debug controller time
	start := time.Now()

	defer func() {
		end := time.Now()
		debug.Info("debug reconcile loop", "durationMs", end.Sub(start).Milliseconds())
	}()
	/////// end debug controller time

	configuration := &servingv1.Configuration{}

	serviceNamespacedName := types.NamespacedName{Name: service.Name, Namespace: service.Namespace}
	configurationNamespacedName := utils.GetConfigurationNamespacedName(serviceNamespacedName)

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
		utils.RemoveEdgeProxyTarget(service)

		return ctrl.Result{}, nil
	}

	debug.Info("compute offload enabled", "name", configurationNamespacedName)

	// retrieve latest traffic split target
	trafficSplit, trafficSplitExists := r.Store.Get(serviceNamespacedName.String())

	if !trafficSplitExists {
		// default to 0
		trafficSplit = 0

		// parse it from annotations
		if service.Annotations != nil {
			previousTrafficStr := service.Annotations[controllers.EdgeProxyTrafficAnnotation]
			previousTraffic, err := strconv.ParseInt(previousTrafficStr, 10, 64)

			if err != nil {
				trafficSplit = previousTraffic
			}
		}
	}

	// override with fixed traffic
	if fixedTraffic, exists := service.Annotations[controllers.EdgeFixedTrafficAnnotation]; exists {
		fixedTrafficSplit, err := strconv.ParseInt(fixedTraffic, 10, 64)

		if err == nil {
			trafficSplit = fixedTrafficSplit
		}
	}

	traffic := make([]servingv1.TrafficTarget, 0, 1)

	// ensure latest target is specified
	if target := utils.GetLatestRevisionTarget(service); target != nil {
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
	if target := utils.GetEdgeProxyTarget(service); target != nil {
		// if the target already exists, check if we need to change the tag

		debug.Info("edge proxy target found", "name", configurationNamespacedName)

		if err := r.Get(ctx, configurationNamespacedName, configuration); err != nil {
			// something is not right, target exists, but revision doesn't
			// requeue after some time
			if apierrors.IsNotFound(err) {
				debug.Info("edge proxy revision not found", "name", configurationNamespacedName)
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			}

			return ctrl.Result{}, err
		}

		debug.Info("debug edge proxy revision", "revision", utils.GetTargetNameFromConfiguration(configuration))

		target.RevisionName = utils.GetTargetNameFromConfiguration(configuration)
		target.Tag = utils.GetTargetTagFromConfiguration(configuration)
		target.Percent = &trafficSplit

		traffic = append(traffic, *target)
	} else {
		// if target revision exists, change the spec
		// this is because we need to change the service if the service
		// is changed in the remote
		debug.Info("edge proxy target not found", "name", configurationNamespacedName)

		if err := r.Get(ctx, configurationNamespacedName, configuration); err != nil {
			if apierrors.IsNotFound(err) {
				debug.Info("edge proxy revision not found", "name", configurationNamespacedName)
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			}

			return ctrl.Result{}, err
		}

		// we know the revision exists but the revision isn't set in the service,
		// so we add it as a traffic target

		debug.Info("debug edge proxy revision", "revision", utils.GetTargetNameFromConfiguration(configuration))

		revisionName := utils.GetTargetNameFromConfiguration(configuration)

		if revisionName == "" {
			// not ready yet, try again later
			debug.Info("edge proxy revision not ready", "name", configurationNamespacedName)
			return ctrl.Result{RequeueAfter: time.Second * 5}, nil
		}

		traffic = append(traffic, servingv1.TrafficTarget{
			RevisionName: revisionName,
			Tag:          utils.GetTargetTagFromConfiguration(configuration),
			Percent:      &trafficSplit,
		})
	}

	// finally, update traffic
	service.Spec.Traffic = traffic

	// update annotations
	annotations := service.Annotations

	if annotations == nil {
		annotations = make(map[string]string)
		service.Annotations = annotations
	}

	annotations[controllers.EdgeProxyTrafficAnnotation] = fmt.Sprint(trafficSplit)

	// logger.Info("debug service", "service", service)

	// requeue every 5 minutes to update the traffic split
	return ctrl.Result{RequeueAfter: strategy.EvaluationPeriodInSeconds * time.Second}, nil
}
