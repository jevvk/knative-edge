package utils

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"edge.jevv.dev/pkg/controllers"
)

func UpdateLastGenerationAnnotation(kind client.Object) {
	annotations := kind.GetAnnotations()

	if annotations == nil {
		annotations = make(map[string]string)
		kind.SetAnnotations(annotations)
	}

	annotations[controllers.LastGenerationAnnotation] = fmt.Sprint(kind.GetResourceVersion())
}

func UpdateLastRemoteGenerationAnnotation(localKind, remoteKind client.Object) {
	annotations := localKind.GetAnnotations()

	if annotations == nil {
		annotations = make(map[string]string)
		localKind.SetAnnotations(annotations)
	}

	annotations[controllers.LastRemoteGenerationAnnotation] = fmt.Sprint(remoteKind.GetResourceVersion())
}

func UpdateLabels(object client.Object) {
	labels := object.GetLabels()

	if labels == nil {
		labels = make(map[string]string)
		object.SetLabels(labels)
	}

	labels[controllers.AppLabel] = "knative-edge"
	labels[controllers.ManagedLabel] = "true"
	labels[controllers.ManagedByLabel] = "knative-edge"
	labels[controllers.CreatedByLabel] = "knative-edge-controller"
}
