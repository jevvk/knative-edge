package utils

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EdgeProxySuffix  = "-edge-proxy"
	EdgeProxyPreffix = "edge-proxy-"
)

func IsEdgeProxyConfiguration(object client.Object) bool {
	if object == nil {
		return false
	}

	return strings.HasSuffix(object.GetName(), EdgeProxySuffix)
}

func IsEdgeProxyRevisionName(name string) bool {
	if len(name) < 6 || strings.Contains(name, "-") {
		return false
	}

	// remove the revision generation (-00000)
	name = string([]rune(name)[:strings.LastIndex(name, "-")])

	return strings.HasSuffix(name, EdgeProxySuffix)
}

func GetConfigurationNamespacedName(namespacedName types.NamespacedName) types.NamespacedName {
	return types.NamespacedName{
		Name:      namespacedName.Name + EdgeProxySuffix,
		Namespace: namespacedName.Namespace,
	}
}

func GetServiceNameFromConfiguration(configuration *servingv1.Configuration) string {
	if !strings.HasSuffix(configuration.Name, EdgeProxySuffix) {
		return ""
	}

	return strings.TrimSuffix(configuration.Name, EdgeProxySuffix)
}

func GetEdgeProxyTarget(service *servingv1.Service) *servingv1.TrafficTarget {
	if service == nil {
		return nil
	}

	for _, target := range service.Spec.Traffic {
		if strings.HasPrefix(target.Tag, EdgeProxyPreffix) {
			return &target
		}
	}

	return nil
}

func RemoveEdgeProxyTarget(service *servingv1.Service) bool {
	if service == nil {
		return false
	}

	hasEdgeProxyTarget := false
	traffic := make([]servingv1.TrafficTarget, 0, len(service.Spec.Traffic))

	for _, target := range service.Spec.Traffic {
		if strings.HasPrefix(target.Tag, EdgeProxyPreffix) {
			hasEdgeProxyTarget = true
			continue
		}

		traffic = append(traffic, target)
	}

	service.Spec.Traffic = traffic

	return hasEdgeProxyTarget
}

func GetLatestRevisionTarget(service *servingv1.Service) *servingv1.TrafficTarget {
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

func GetConfigurationGenerationFromTarget(target *servingv1.TrafficTarget) string {
	if target == nil {
		return ""
	}

	generation := strings.TrimPrefix(target.Tag, EdgeProxyPreffix)
	generation = strings.TrimPrefix("0", generation)

	return generation
}

func GetTargetNameFromConfiguration(configuration *servingv1.Configuration) string {
	return configuration.Status.LatestReadyRevisionName
}

func GetTargetTagFromConfiguration(configuration *servingv1.Configuration) string {
	generation := -1

	if configuration != nil {
		generation = int(configuration.GetGeneration())
	}

	return fmt.Sprintf("%s%05d", EdgeProxyPreffix, generation)
}
