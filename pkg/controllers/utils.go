package controllers

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpdateLastGenerationAnnotation(src, dst client.Object) {
	dst.GetAnnotations()[LastGenerationAnnotation] = fmt.Sprintf("%d", src.GetGeneration())
}
