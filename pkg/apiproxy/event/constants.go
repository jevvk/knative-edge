package event

import (
	"encoding/json"
	"errors"
)

type EncodingType int

const (
	JsonEncoding EncodingType = iota
	CompressedEncodingV1
)

const (
	jsonEnconding       string = "json"
	compressedEnconding string = "compressed-v1"
)

const DefaultEncoding = JsonEncoding

func (enum *EncodingType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	switch s {
	case jsonEnconding:
		*enum = JsonEncoding
	case compressedEnconding:
		*enum = CompressedEncodingV1
	default:
		return errors.New("invalid EncodingType")
	}

	return nil
}

func (enum EncodingType) String() string {
	var s string

	switch enum {
	case JsonEncoding:
		s = jsonEnconding
	case CompressedEncodingV1:
		s = compressedEnconding
	}

	return s
}

func (enum EncodingType) MarshalJSON() ([]byte, error) {
	return json.Marshal(enum.String())
}

type EventType int

const (
	SyncEventType EventType = iota

	NamespaceCreateEventType
	NamespaceDeleteEventType
	NamespaceUpdateEventType

	ResourceDeleteEventType
	ResourceCreateEventType
	ResourceUpdateEventType
	ResourceBatchEventType
)

const (
	syncEventType string = "sync"

	namespaceCreateEventType string = "namespace-create"
	namespaceDeleteEventType string = "namespace-delete"
	namespaceUpdateEventType string = "namespace-update"

	resourceDeleteEventType string = "resource-delete"
	resourceCreateEventType string = "resource-create"
	resourceUpdateEventType string = "resource-update"
	resourceBatchEventType  string = "resource-batch"
)

func (enum *EventType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	switch s {
	case resourceBatchEventType:
		*enum = ResourceBatchEventType
	case resourceUpdateEventType:
		*enum = ResourceUpdateEventType
	case resourceCreateEventType:
		*enum = ResourceCreateEventType
	case resourceDeleteEventType:
		*enum = ResourceDeleteEventType

	case namespaceUpdateEventType:
		*enum = NamespaceUpdateEventType
	case namespaceCreateEventType:
		*enum = NamespaceCreateEventType
	case namespaceDeleteEventType:
		*enum = NamespaceDeleteEventType

	case syncEventType:
		*enum = SyncEventType

	default:
		return errors.New("invalid EventType")
	}

	return nil
}

func (enum EventType) String() string {
	var s string

	switch enum {
	case ResourceBatchEventType:
		s = resourceBatchEventType
	case ResourceUpdateEventType:
		s = resourceUpdateEventType
	case ResourceCreateEventType:
		s = resourceCreateEventType
	case ResourceDeleteEventType:
		s = resourceDeleteEventType

	case NamespaceUpdateEventType:
		s = namespaceUpdateEventType
	case NamespaceCreateEventType:
		s = namespaceCreateEventType
	case NamespaceDeleteEventType:
		s = namespaceDeleteEventType

	case SyncEventType:
		s = syncEventType
	}

	return s
}

func (enum EventType) MarshalJSON() ([]byte, error) {
	return json.Marshal(enum.String())
}
