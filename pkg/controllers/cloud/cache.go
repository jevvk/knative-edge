package cloud

import (
	klabels "k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	"edge.jevv.dev/pkg/controllers"
)

var ScopedCache = cache.BuilderWithOptions(cache.Options{
	DefaultSelector: cache.ObjectSelector{
		Label: klabels.SelectorFromSet(map[string]string{
			controllers.ManagedLabel: "true",
		}),
	},
})
