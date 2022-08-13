package controllers

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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

	oldGeneration := fmt.Sprintf("%d", e.ObjectOld.GetGeneration())
	newGeneration := e.ObjectNew.GetAnnotations()[LastGenerationAnnotation]

	return oldGeneration != newGeneration
}
