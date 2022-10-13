package edge

import (
	"fmt"

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

// Update implements default UpdateEvent filter for validating generation change.
func (NotChangedByEdgeControllers) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		return false
	}

	if e.ObjectNew == nil {
		return false
	}

	oldGeneration := fmt.Sprint(e.ObjectOld.GetGeneration())
	newGeneration := e.ObjectNew.GetAnnotations()[controllers.LastGenerationAnnotation]

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

func IsManagedObject(obj client.Object) bool {
	labels := obj.GetLabels()

	if labels == nil {
		return false
	}

	for label, value := range labels {
		if label != controllers.ManagedLabel {
			continue
		}

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
					Key:      controllers.AppLabel,
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"knative-edge"},
				},
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
					Key:      controllers.AppLabel,
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"knative-edge"},
				},
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
