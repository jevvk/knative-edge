package edge

import (
	klabels "k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	corev1 "k8s.io/api/core/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"edge.knative.dev/pkg/labels"
)

var ScopedCache = cache.BuilderWithOptions(cache.Options{
	SelectorsByObject: cache.SelectorsByObject{
		&corev1.Secret{}: cache.ObjectSelector{
			Label: klabels.SelectorFromSet(map[string]string{
				labels.ManagedLabel: "true",
			}),
		},
		&corev1.ConfigMap{}: cache.ObjectSelector{
			Label: klabels.SelectorFromSet(map[string]string{
				labels.ManagedLabel: "true",
			}),
		},
		&servingv1.Service{}: cache.ObjectSelector{
			Label: klabels.SelectorFromSet(map[string]string{
				labels.ManagedLabel: "true",
			}),
		},
	},
})
