package controllers

import (
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	corev1 "k8s.io/api/core/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

var ScopedCache = cache.BuilderWithOptions(cache.Options{
	SelectorsByObject: cache.SelectorsByObject{
		&corev1.Secret{}: cache.ObjectSelector{
			Field: fields.SelectorFromSet(map[string]string{
				".metadata.labels.['edge\\.knative\\.dev/synchronize']": "true",
			}),
		},
		&corev1.ConfigMap{}: cache.ObjectSelector{
			Field: fields.SelectorFromSet(map[string]string{
				".metadata.labels.['edge\\.knative\\.dev/synchronize']": "true",
			}),
		},
		&servingv1.Service{}: cache.ObjectSelector{
			Field: fields.SelectorFromSet(map[string]string{
				".metadata.labels.['edge\\.knative\\.dev/synchronize']": "true",
			}),
		},
	},
})
