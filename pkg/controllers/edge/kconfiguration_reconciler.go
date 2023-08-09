package edge

import (
	"context"
	"fmt"
	"reflect"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"

	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"edge.jevv.dev/pkg/controllers"
	"edge.jevv.dev/pkg/controllers/utils"

	corev1 "k8s.io/api/core/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

func (r *KServiceReconciler) reconcileKConfiguration(ctx context.Context, service *servingv1.Service) (ctrl.Result, error) {
	log := r.Log.WithName("configuration").V(controllers.InfoLevel)
	debug := r.Log.WithName("configuration").V(controllers.DebugLevel)

	if service == nil {
		debug.Info("no service")
		return ctrl.Result{}, nil
	}

	//////// debug controller time
	start := time.Now()

	defer func() {
		end := time.Now()
		debug.Info("debug reconcile loop", "durationMs", end.Sub(start).Milliseconds())
	}()
	/////// end debug controller time

	if service.ResourceVersion == "" {
		debug.Info("no local service")
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	shouldCreate := false
	shouldUpdate := false
	shouldDelete := false

	localConfiguration := &servingv1.Configuration{}

	serviceNamespacedName := types.NamespacedName{Name: service.Name, Namespace: service.Namespace}
	configurationNamespacedName := utils.GetConfigurationNamespacedName(serviceNamespacedName)

	if err := r.Get(ctx, configurationNamespacedName, localConfiguration); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		shouldCreate = true
	}

	configuration := localConfiguration.DeepCopy()

	if !kServiceHasComputeOffloadLabel(service) {
		if shouldCreate {
			// doesn't exist and doesn't need to exist, can exit early
			return ctrl.Result{}, nil
		}

		shouldDelete = true
		shouldCreate = false
	}

	annotations := configuration.Annotations

	if annotations == nil {
		annotations = make(map[string]string)
		configuration.Annotations = annotations
	}

	// TODO: check if we should update
	if !shouldDelete {
		r.buildConfiguration(configurationNamespacedName, configuration, service)

		shouldUpdate = !reflect.DeepEqual(localConfiguration, configuration)

		// do this after, otherwise we're stuck in infinite loop
		utils.UpdateLastRemoteGenerationAnnotation(configuration, service)

		// don't actually wanna set as owner, otherwise it interferes with knative
		// controllerutil.SetControllerReference(service, configuration, r.Scheme)
	}

	debug.Info("debug bool", "shouldCreate", shouldCreate, "shouldUpdate", shouldUpdate, "shouldDelete", shouldDelete)
	debug.Info("debug name", "configurationName", configurationNamespacedName)

	if shouldCreate {
		log.Info("Creating edge proxy route.", "name", configurationNamespacedName)

		if err := r.Create(ctx, configuration); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			debug.Error(err, "couldn't create configuration")

			return ctrl.Result{}, err
		}
	} else if shouldUpdate {
		log.Info("Updating edge proxy route.", "name", configurationNamespacedName)

		if err := r.Update(ctx, configuration); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{}, err
		}
	} else if shouldDelete {
		log.Info("Deleting edge proxy route.", "name", configurationNamespacedName)

		if err := r.Delete(ctx, configuration); err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}

			return ctrl.Result{}, err
		}
	}

	if shouldCreate || shouldUpdate {
		// requeue after in order to update service
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (r *KServiceReconciler) buildConfiguration(namespacedName types.NamespacedName, configuration *servingv1.Configuration, service *servingv1.Service) {
	serviceAnnotations := service.Annotations

	if serviceAnnotations == nil {
		serviceAnnotations = make(map[string]string)
	}

	annotations := configuration.Annotations

	if annotations == nil {
		annotations = make(map[string]string)
		configuration.Annotations = annotations
	}

	labels := configuration.Labels

	if labels == nil {
		labels = make(map[string]string)
		configuration.Labels = labels
	}

	configuration.Name = namespacedName.Name
	configuration.Namespace = namespacedName.Namespace

	labels[controllers.EdgeLocalLabel] = "true"
	labels[controllers.ManagedLabel] = "true"
	labels[controllers.CreatedByLabel] = "knative-edge"
	labels[controllers.ManagedByLabel] = "knative-edge"
	// labels[controllers.KServiceLabel] = service.Name
	// labels[controllers.KServiceUIDLabel] = string(service.UID)

	annotations[controllers.LastGenerationAnnotation] = fmt.Sprint(service.Generation)
	// annotations[controllers.KnativeNoGCAnnotation] = "true"

	// need to be careful not to override immutable values from configuration

	containers := configuration.Spec.Template.Spec.PodSpec.Containers

	if containers == nil {
		containers = make([]corev1.Container, 0, 1)
		configuration.Spec.Template.Spec.PodSpec.Containers = containers
	}

	if len(containers) == 0 {
		containers = append(containers, corev1.Container{})
		configuration.Spec.Template.Spec.PodSpec.Containers = containers
	}

	var concurrency int64 = 8
	configuration.Spec.Template.Spec.ContainerConcurrency = &concurrency

	container := &containers[0]

	// TODO: set concurrency and timeout
	container.Name = "edge-proxy"
	container.Image = r.ProxyImage
	container.Env = []corev1.EnvVar{
		{Name: "REMOTE_URL", Value: serviceAnnotations[controllers.RemoteUrlAnnotation]},
		{Name: "REMOTE_HOST", Value: serviceAnnotations[controllers.RemoteHostAnnotation]},
		{Name: "REMOTE_PROXY", Value: r.HttpProxy},
		// {Name: "HTTP_PROXY", Value: r.HttpProxy},
		// {Name: "HTTPS_PROXY", Value: r.HttpsProxy},
		// {Name: "NO_PROXY", Value: r.NoProxy},
	}

	specLabels := configuration.Spec.Template.Labels

	if specLabels == nil {
		specLabels = make(map[string]string)
		configuration.Spec.Template.Labels = specLabels
	}

	specLabels[controllers.EdgeLocalLabel] = "true"
	specLabels[controllers.ManagedLabel] = "true"

	// experiments
	specAnnotations := configuration.Spec.Template.Annotations

	if specAnnotations == nil {
		specAnnotations = make(map[string]string)
		configuration.Spec.Template.Annotations = specAnnotations
	}

	specAnnotations["autoscaling.knative.dev/min-scale"] = "3"
	specAnnotations["autoscaling.knative.dev/max-scale"] = "5"
}
