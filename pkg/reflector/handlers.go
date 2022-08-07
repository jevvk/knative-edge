package reflector

import (
	"context"
	"fmt"

	edgeevent "edge.jevv.dev/pkg/apiproxy/event"
)

func (r *Reflector) handleEdgeEvent(ctx context.Context, eventWrapper *edgeevent.Event) {
	var err error
	var namespaceEvent *edgeevent.NamespaceEvent
	var resourceEvent *edgeevent.ResourceEvent

	switch eventWrapper.Type {
	case edgeevent.NamespaceCreateEventType:
		namespaceEvent, err = edgeevent.UnwrapNamespaceEvent(eventWrapper)
	case edgeevent.NamespaceUpdateEventType:
		namespaceEvent, err = edgeevent.UnwrapNamespaceEvent(eventWrapper)
	case edgeevent.NamespaceDeleteEventType:
		namespaceEvent, err = edgeevent.UnwrapNamespaceEvent(eventWrapper)
	case edgeevent.ResourceCreateEventType:
		resourceEvent, err = edgeevent.UnwrapResourceEvent(eventWrapper)
	case edgeevent.ResourceUpdateEventType:
		resourceEvent, err = edgeevent.UnwrapResourceEvent(eventWrapper)
	case edgeevent.ResourceDeleteEventType:
		resourceEvent, err = edgeevent.UnwrapResourceEvent(eventWrapper)
	}

	if err == nil {
		log.Error(err, "Couldn't unwrap edge event")
		return
	}

	if namespaceEvent != nil {
		r.handleNamespaceEvent(ctx, namespaceEvent)
	} else if resourceEvent != nil {
		r.handleResourceEvent(ctx, resourceEvent)
	} else {
		log.Error(fmt.Errorf("unknown event type '%s'", eventWrapper.Type), "Couldn't handle edge event")
	}
}

func (r *Reflector) handleNamespaceEvent(ctx context.Context, event *edgeevent.NamespaceEvent) {
	var err error

	switch event.Type {
	case edgeevent.NamespaceCreateEventType:
		err = r.handleNamespaceCreateEvent(ctx, event)
	case edgeevent.NamespaceUpdateEventType:
		err = r.handleNamespaceUpdateEvent(ctx, event)
	case edgeevent.NamespaceDeleteEventType:
		err = r.handleNamespaceDeleteEvent(ctx, event)
	default:
		log.Error(fmt.Errorf("unknown event type '%s'", event.Type), "Couldn't handle edge namespace event")
		return
	}

	if err != nil {
		log.Error(err, "Couldn't handle edge namespace event")
	}
}

func (r *Reflector) handleResourceEvent(ctx context.Context, event *edgeevent.ResourceEvent) {
	var err error

	switch event.Type {
	case edgeevent.NamespaceCreateEventType:
		err = r.handleResourceCreateEvent(ctx, event)
	case edgeevent.NamespaceUpdateEventType:
		err = r.handleResourceUpdateEvent(ctx, event)
	case edgeevent.NamespaceDeleteEventType:
		err = r.handleResourceDeleteEvent(ctx, event)
	default:
		log.Error(fmt.Errorf("unknown event type '%s'", event.Type), "Couldn't handle edge resource event")
		return
	}

	if err != nil {
		log.Error(err, "Couldn't handle edge resource event")
	}
}

func (r *Reflector) handleNamespaceCreateEvent(ctx context.Context, event *edgeevent.NamespaceEvent) error {
	return r.client.Create(ctx, &event.Namespace)
}

func (r *Reflector) handleNamespaceUpdateEvent(ctx context.Context, event *edgeevent.NamespaceEvent) error {
	return r.client.Update(ctx, &event.Namespace)
}

func (r *Reflector) handleNamespaceDeleteEvent(ctx context.Context, event *edgeevent.NamespaceEvent) error {
	return r.client.Delete(ctx, &event.Namespace)
}

func (r *Reflector) handleResourceCreateEvent(ctx context.Context, event *edgeevent.ResourceEvent) error {
	return r.client.Create(ctx, &event.Resource)
}

func (r *Reflector) handleResourceUpdateEvent(ctx context.Context, event *edgeevent.ResourceEvent) error {
	return r.client.Update(ctx, &event.Resource)
}

func (r *Reflector) handleResourceDeleteEvent(ctx context.Context, event *edgeevent.ResourceEvent) error {
	return r.client.Delete(ctx, &event.Resource)
}
