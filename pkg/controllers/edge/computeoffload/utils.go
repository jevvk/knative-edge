package computeoffload

import (
	"strings"

	"k8s.io/apimachinery/pkg/types"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	suffix = "-edge-compute-offload"
	tag    = "edge-compute-offload"
)

func isComputeOffloading(object client.Object) bool {
	if object == nil {
		return false
	}

	return strings.HasSuffix(object.GetName(), suffix)
}

func getRevisionNamespacedName(namespacedName types.NamespacedName) types.NamespacedName {
	return types.NamespacedName{
		Name:      namespacedName.Name + suffix,
		Namespace: namespacedName.Namespace,
	}
}

func getComputeOffloadTrafficTarget(service *servingv1.Service) *servingv1.TrafficTarget {
	if service == nil {
		return nil
	}

	for _, target := range service.Spec.Traffic {
		if target.Tag == tag {
			return &target
		}
	}

	return nil
}
