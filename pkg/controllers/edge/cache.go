package edge

import (
	"fmt"
	// "os"

	klabels "k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	// corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"edge.jevv.dev/pkg/controllers"
)

func EnvScopedCache(envs []string) cache.NewCacheFunc {
	var labelSelector metav1.LabelSelector

	if len(envs) == 0 {
		labelSelector = metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
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
	SelectorsByObject: cache.SelectorsByObject{
		// note: not sure why I wanna watch pods
		// &corev1.Pod{}: cache.ObjectSelector{
		// 	Label: klabels.SelectorFromSet(map[string]string{
		// 		controllers.AppLabel:     "knative-edge",
		// 		controllers.EdgeTagLabel: os.Getenv("EDGE_DEPLOYMENT_TAG"),
		// 	}),
		// },
		// only watch for configurations managed by controller
		&servingv1.Configuration{}: cache.ObjectSelector{
			Label: klabels.SelectorFromSet(map[string]string{
				controllers.ManagedLabel:   "true",
				controllers.EdgeLocalLabel: "true",
			}),
		},
	},
})
