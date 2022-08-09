package edge

import (
	"fmt"
	"strings"

	klabels "k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	"edge.jevv.dev/pkg/labels"
)

func EnvScopedCache(envs []string) cache.NewCacheFunc {
	selector, err := klabels.Parse(fmt.Sprintf("%s in (%s)", labels.EnvironmentLabel, strings.Join(envs, ",")))

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
			labels.ManagedLabel: "true",
		}),
	},
})
