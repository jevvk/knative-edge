package event

import (
	edgev1 "edge.knative.dev/pkg/apis/edge/v1"
)

type NamespaceEvent struct {
	Type      EventType            `json:"type"`
	Namespace edgev1.EdgeNamespace `json:"namespace"`
}

func UnwrapNamespaceEvent(e *Event) (*NamespaceEvent, error) {
	var namespace edgev1.EdgeNamespace
	err := Decode(e.Encoding, []byte(e.Data), &namespace)

	if err != nil {
		return nil, err
	}

	return &NamespaceEvent{
		Type:      e.Type,
		Namespace: namespace,
	}, nil
}

func WrapNamespaceEvent(e *NamespaceEvent) (*Event, error) {
	return WrapEvent(e.Type, e)
}
