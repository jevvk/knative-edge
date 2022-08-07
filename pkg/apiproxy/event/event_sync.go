package event

import (
	"errors"
	"fmt"

	edgev1 "edge.jevv.dev/pkg/apis/edge/v1"
)

type SyncEvent struct {
	immutable bool `json:"-"`

	Namespace string                  `json:"namespace"`
	Resources map[string]resourceSync `json:"resources"`
}

func NewSyncEvent(namespace string) *SyncEvent {
	return &SyncEvent{
		immutable: false,
		Namespace: namespace,
		Resources: make(map[string]resourceSync),
	}
}

func (e *SyncEvent) AddResource(resource *edgev1.EdgeResource) error {
	if e.immutable {
		return errors.New("sync event is immutable")
	}

	resourceName := fmt.Sprintf("%s/%s", resource.Spec.ApiVersion, resource.Spec.Kind)

	e.Resources[resourceName] = resourceSync{
		Name:                  resource.Spec.Name,
		ApiVersion:            resource.Spec.ApiVersion,
		Kind:                  resource.Spec.Kind,
		LocalResourceVersion:  resource.Status.ResourceVersion,
		RemoteResourceVersion: resource.Spec.RemoteResourceVersion,
	}

	return nil
}

func UnwrapSyncEvent(e *Event) (*SyncEvent, error) {
	if e.Type != SyncEventType {
		return nil, errors.New("not a sync event")
	}

	var resources map[string]resourceSync
	err := Decode(e.Encoding, []byte(e.Data), &resources)

	if err != nil {
		return nil, err
	}

	return &SyncEvent{
		immutable: true,
		Resources: resources,
	}, nil
}

type resourceSync struct {
	Name                  string
	ApiVersion            string
	Kind                  string
	Namespace             string
	LocalResourceVersion  string
	RemoteResourceVersion string
}
