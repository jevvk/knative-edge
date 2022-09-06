package edge

import (
	"fmt"
	"strings"

	klabels "k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	"edge.jevv.dev/pkg/controllers"
)

func EnvScopedCache(envs []string) cache.NewCacheFunc {
	var err error
	var selector klabels.Selector

	if len(envs) == 0 {
		selector, err = klabels.Parse(controllers.EnvironmentLabel)
	} else {
		selector, err = klabels.Parse(fmt.Sprintf("%s in (%s)", controllers.EnvironmentLabel, strings.Join(envs, ",")))
	}

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
