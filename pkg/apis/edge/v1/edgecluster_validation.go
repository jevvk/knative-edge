package v1

import (
	"context"

	"knative.dev/pkg/apis"
)

func (ec *EdgeCluster) Validate(ctx context.Context) (errs *apis.FieldError) {
	return ec.Spec.Validate(apis.WithinSpec(ctx))
}

func (ecs *EdgeClusterSpec) Validate(ctx context.Context) (errs *apis.FieldError) {
	return nil
}
