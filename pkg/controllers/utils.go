package controllers

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpdateLastGenerationAnnotation(src, dst client.Object) {
	annotations := dst.GetAnnotations()

	if annotations == nil {
		annotations = make(map[string]string)
		dst.SetAnnotations(annotations)
	}

	annotations[LastGenerationAnnotation] = fmt.Sprintf("%d", src.GetGeneration())
}

func UpdateLabels(object client.Object) {
	labels := object.GetLabels()

	if labels == nil {
		labels = make(map[string]string)
		object.SetAnnotations(labels)
	}

	labels[ManagedLabel] = "true"
	labels[ManagedByLabel] = "knative-edge"
	labels[CreatedByLabel] = "knative-edge-controller"
}
