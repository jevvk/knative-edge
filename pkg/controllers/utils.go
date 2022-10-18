package controllers

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpdateLastGenerationAnnotation(kind client.Object) {
	annotations := kind.GetAnnotations()

	if annotations == nil {
		annotations = make(map[string]string)
		kind.SetAnnotations(annotations)
	}

	annotations[LastGenerationAnnotation] = fmt.Sprint(kind.GetResourceVersion())
}

func UpdateLastRemoteGenerationAnnotation(localKind, remoteKind client.Object) {
	annotations := localKind.GetAnnotations()

	if annotations == nil {
		annotations = make(map[string]string)
		localKind.SetAnnotations(annotations)
	}

	annotations[LastRemoteGenerationAnnotation] = fmt.Sprint(remoteKind.GetResourceVersion())
}

func UpdateLabels(object client.Object) {
	labels := object.GetLabels()

	if labels == nil {
		labels = make(map[string]string)
		object.SetLabels(labels)
	}

	labels[AppLabel] = "knative-edge"
	labels[ManagedLabel] = "true"
	labels[ManagedByLabel] = "knative-edge"
	labels[CreatedByLabel] = "knative-edge-controller"
}
