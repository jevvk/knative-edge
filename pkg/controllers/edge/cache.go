package edge

import (
	"fmt"

	klabels "k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"edge.jevv.dev/pkg/controllers"
)

func EnvScopedCache(envs []string) cache.NewCacheFunc {
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
		panic(fmt.Errorf("couldn't create label selector: %w", err))
	}

	return cache.BuilderWithOptions(cache.Options{
		DefaultSelector: cache.ObjectSelector{
			Label: selector,
		},
	})
}

var ManagedScopedCache = cache.BuilderWithOptions(cache.Options{
	DefaultSelector: cache.ObjectSelector{
		Label: klabels.SelectorFromSet(map[string]string{
			controllers.ManagedLabel: "true",
		}),
	},
})
