package edge

import (
	"fmt"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"edge.jevv.dev/pkg/controllers"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type NotChangedByEdgeControllers struct {
	predicate.Funcs
}

func (NotChangedByEdgeControllers) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}

	annotations := e.Object.GetAnnotations()

	if annotations == nil {
		return false
	}

	// by default, it's 0 when create by edge controller
	return annotations[controllers.LastGenerationAnnotation] != "0"
}

func (NotChangedByEdgeControllers) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		return false
	}

	if e.ObjectNew == nil {
		return false
	}

	annotations := e.ObjectNew.GetAnnotations()

	if annotations == nil {
		return false
	}

	oldGeneration := fmt.Sprint(e.ObjectOld.GetResourceVersion())
	newGeneration := annotations[controllers.LastGenerationAnnotation]

	return oldGeneration != newGeneration
}

func HasEdgeSyncLabel(obj client.Object, envs []string) bool {
	if envs == nil {
		return false
	}

	labels := obj.GetLabels()

	if labels == nil {
		return false
	}

	for label, value := range labels {
		if label != controllers.EnvironmentLabel {
			continue
		}

		for _, env := range envs {
			if value == env {
				return true
			}
		}

		return false
	}

	return false
}

var IsManagedByEdgeControllers = predicate.NewPredicateFuncs(IsManagedObject)

func IsManagedObject(obj client.Object) bool {
	labels := obj.GetLabels()

	if labels == nil {
		return false
	}

	if value, ok := labels[controllers.ManagedLabel]; ok {
		return value == "true"
	}

	return false
}

func HasEdgeSyncLabelPredicate(envs []string) predicate.Predicate {
	var labelSelector metav1.LabelSelector

	if len(envs) == 0 {
		labelSelector = metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      controllers.EnvironmentLabel,
					Operator: metav1.LabelSelectorOpExists,
				},
			},
		}
	} else {
		labelSelector = metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      controllers.EnvironmentLabel,
					Operator: metav1.LabelSelectorOpIn,
					Values:   envs,
				},
			},
		}
	}

	selector, err := metav1.LabelSelectorAsSelector(&labelSelector)

	if err != nil {
		panic(fmt.Errorf("couldn't create label selector predicate: %w", err))
	}

	filter := func(o client.Object) bool {
		if o == nil {
			return false
		}

		return selector.Matches(labels.Set(o.GetLabels()))
	}

	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return filter(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return filter(e.ObjectOld) || filter(e.ObjectNew)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return filter(e.Object)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return filter(e.Object)
		},
	}
}

type LatestReadyRevisionChangedPredicate struct {
	predicate.Funcs
}

func (LatestReadyRevisionChangedPredicate) Update(e event.UpdateEvent) bool {
	var ok bool
	var oldConfiguration *servingv1.Configuration
	var newConfiguration *servingv1.Configuration

	if e.ObjectOld == nil {
		return e.ObjectNew != nil
	}

	if oldConfiguration, ok = e.ObjectOld.(*servingv1.Configuration); !ok {
		return false
	}

	if newConfiguration, ok = e.ObjectNew.(*servingv1.Configuration); !ok {
		return false
	}

	return oldConfiguration.Status.LatestReadyRevisionName != newConfiguration.Status.LatestReadyRevisionName
}

type RemoteResourceChangedPredicate struct {
	predicate.Funcs
}

func haveRemoteResourceLabelsChanged(objectOld, objectNew client.Object) bool {
	labelsOld := objectOld.GetLabels()

	if labelsOld == nil {
		labelsOld = make(map[string]string)
	}

	labelsNew := objectNew.GetLabels()

	if labelsNew == nil {
		labelsNew = make(map[string]string)
	}

	return labelsOld[controllers.EnvironmentLabel] != labelsNew[controllers.EnvironmentLabel] ||
		labelsOld[controllers.EdgeOffloadLabel] != labelsNew[controllers.EdgeOffloadLabel]
}

func (RemoteResourceChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		return true
	}

	if e.ObjectNew == nil {
		return false
	}

	// spec has changed
	if e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() {
		return true
	}

	// labels have changed
	if haveRemoteResourceLabelsChanged(e.ObjectOld, e.ObjectNew) {
		return true
	}

	return false
}
