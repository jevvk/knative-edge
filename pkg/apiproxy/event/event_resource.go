package event

import (
	edgev1 "edge.knative.dev/pkg/apis/edge/v1"
)

type ResourceEvent struct {
	Type     EventType           `json:"type"`
	Resource edgev1.EdgeResource `json:"resource"`
}

func UnwrapResourceEvent(e *Event) (*ResourceEvent, error) {
	var resource edgev1.EdgeResource
	err := Decode(e.Encoding, []byte(e.Data), &resource)

	if err != nil {
		return nil, err
	}

	return &ResourceEvent{
		Type:     e.Type,
		Resource: resource,
	}, nil
}

func WrapResourceEvent(e *ResourceEventBatch) (*Event, error) {
	return WrapEvent(e.Type, e)
}

type ResourceEventBatch struct {
	Type  EventType
	Batch []ResourceEvent
}

func UnwrapResourceEventBatch(e *Event) (*ResourceEventBatch, error) {
	var batch []ResourceEvent
	err := Decode(e.Encoding, []byte(e.Data), &batch)

	if err != nil {
		return nil, err
	}

	return &ResourceEventBatch{
		Type:  e.Type,
		Batch: batch,
	}, nil
}

func WrapResourceEventBatch(e *ResourceEventBatch) (*Event, error) {
	return WrapEvent(e.Type, e)
}
