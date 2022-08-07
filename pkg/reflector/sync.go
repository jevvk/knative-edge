package reflector

import (
	"context"
	"errors"
	"fmt"
	"time"

	"edge.jevv.dev/pkg/apiproxy/event"
	edgev1 "edge.jevv.dev/pkg/apis/edge/v1"
)

func (r *Reflector) sync(ctx context.Context) error {
	if r.edgeClient == nil {
		return errors.New("no edge client found")
	}

	var namespaceList edgev1.EdgeNamespaceList

	if err := r.client.List(ctx, &namespaceList); err != nil {
		return fmt.Errorf("couldn't list edge namespaces: %s", err)
	}

	for _, namespace := range namespaceList.Items {
		var resourceList edgev1.EdgeResourceList

		if err := r.client.List(ctx, &resourceList); err != nil {
			return fmt.Errorf("couldn't list edge resoures: %s", err)
		}

		syncEvent := event.NewSyncEvent(namespace.Name)

		for _, resource := range resourceList.Items {
			// no need to check for error since it's not immutable
			syncEvent.AddResource(&resource)
		}

		// TODO: push sync event
	}

	return nil
}

func (r *Reflector) Sync(ctx context.Context) error {
	go func() {
		for {
			err := r.sync(ctx)

			if err != nil {
				log.Error(err, "Couldn't sync edge resources.")
			}

			time.Sleep(r.SyncPeriod)
		}
	}()

	<-ctx.Done()
	return nil
}
