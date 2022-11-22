package edge

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nameSuffix = "-edge-proxy"
	tagPreffix = "edge-proxy-"
)

func isEdgeProxyConfiguration(object client.Object) bool {
	if object == nil {
		return false
	}

	return strings.HasSuffix(object.GetName(), nameSuffix)
}

func getConfigurationNamespacedName(namespacedName types.NamespacedName) types.NamespacedName {
	return types.NamespacedName{
		Name:      namespacedName.Name + nameSuffix,
		Namespace: namespacedName.Namespace,
	}
}

func getServiceNameFromConfiguration(configuration *servingv1.Configuration) string {
	if !strings.HasSuffix(configuration.Name, nameSuffix) {
		return ""
	}

	return strings.TrimSuffix(configuration.Name, nameSuffix)
}

func getEdgeProxyTarget(service *servingv1.Service) *servingv1.TrafficTarget {
	if service == nil {
		return nil
	}

	for _, target := range service.Spec.Traffic {
		if strings.HasPrefix(target.Tag, tagPreffix) {
			return &target
		}
	}

	return nil
}

func removeEdgeProxyTarget(service *servingv1.Service) bool {
	if service == nil {
		return false
	}

	hasEdgeProxyTarget := false
	traffic := make([]servingv1.TrafficTarget, 0, len(service.Spec.Traffic))

	for _, target := range service.Spec.Traffic {
		if strings.HasPrefix(target.Tag, tagPreffix) {
			hasEdgeProxyTarget = true
			continue
		}

		traffic = append(traffic, target)
	}

	service.Spec.Traffic = traffic

	return hasEdgeProxyTarget
}

func getLatestRevisionTarget(service *servingv1.Service) *servingv1.TrafficTarget {
	if service == nil {
		return nil
	}

	for _, target := range service.Spec.Traffic {
		if target.LatestRevision != nil && *target.LatestRevision {
			return &target
		}
	}

	return nil
}

func getConfigurationGenerationFromTarget(target *servingv1.TrafficTarget) string {
	if target == nil {
		return ""
	}

	generation := strings.TrimPrefix(target.Tag, tagPreffix)
	generation = strings.TrimPrefix("0", generation)

	return generation
}

func getTargetNameFromConfiguration(configuration *servingv1.Configuration) string {
	return configuration.Status.LatestReadyRevisionName
}

func getTargetTagFromConfiguration(configuration *servingv1.Configuration) string {
	generation := -1

	if configuration != nil {
		generation = int(configuration.GetGeneration())
	}

	return fmt.Sprintf("%s%05d", tagPreffix, generation)
}
