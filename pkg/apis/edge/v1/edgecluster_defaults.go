package v1

import (
	"context"

	"knative.dev/pkg/apis"
)

func (ec *EdgeCluster) SetDefaults(ctx context.Context) {
	ec.Spec.SetDefaults(apis.WithinSpec(ctx))
}

func (ecs *EdgeClusterSpec) SetDefaults(ctx context.Context) {

}
