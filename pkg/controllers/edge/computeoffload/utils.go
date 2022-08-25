package computeoffload

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nameSuffix = "-edge-compute-offload"
	tagPreffix = "edge-compute-offload-"
)

func isComputeOffloadRevision(object client.Object) bool {
	if object == nil {
		return false
	}

	return strings.HasSuffix(object.GetName(), nameSuffix)
}

func getRevisionNamespacedName(namespacedName types.NamespacedName) types.NamespacedName {
	return types.NamespacedName{
		Name:      namespacedName.Name + nameSuffix,
		Namespace: namespacedName.Namespace,
	}
}

func getComputeOffloadTarget(service *servingv1.Service) *servingv1.TrafficTarget {
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

func getLatestResivionTarget(service *servingv1.Service) *servingv1.TrafficTarget {
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

func getRevisionGenerationFromTarget(target *servingv1.TrafficTarget) string {
	if target == nil {
		return ""
	}

	return strings.TrimPrefix(target.Tag, tagPreffix)
}

func getTargetTagFromRevision(revision *servingv1.Revision) string {
	generation := -1

	if revision != nil {
		generation = int(revision.GetGeneration())
	}

	return fmt.Sprintf("%s%d", tagPreffix, generation)
}
