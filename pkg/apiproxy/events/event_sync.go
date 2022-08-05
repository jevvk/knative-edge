package events

import (
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"edge.knative.dev/pkg/labels"
)

const SyncEventType = "sync"

type SyncEvent struct {
	immutable bool

	Namespace string
	Resources map[string]resourceSync
}

func (SyncEvent) New(namespace string) *SyncEvent {
	return &SyncEvent{
		immutable: false,
		Namespace: namespace,
		Resources: make(map[string]resourceSync),
	}
}

func (e *SyncEvent) AddResource(obj client.Object) error {
	if e.immutable {
		return errors.New("sync event is immutable")
	}

	remoteResourceVersion := obj.GetLabels()[labels.RemoteResourceVersionLabel]

	kind := obj.GetObjectKind().GroupVersionKind()
	apiVersion := fmt.Sprintf("%s/%s", kind.Group, kind.Version)
	resourceName := fmt.Sprintf("%s/%s/%s", kind.Group, kind.Version, kind.Kind)

	e.Resources[resourceName] = resourceSync{
		Name:                  obj.GetName(),
		ApiVersion:            apiVersion,
		Kind:                  kind.Kind,
		LocalResourceVersion:  obj.GetResourceVersion(),
		RemoteResourceVersion: remoteResourceVersion,
	}

	return nil
}

func (SyncEvent) NewFromEvent(e *Event) (*SyncEvent, error) {
	if e.Type != SyncEventType {
		return nil, errors.New("not a sync event")
	}

	var resources map[string]resourceSync
	err := Decode(e.Encoding, e.Data, &resources)

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
