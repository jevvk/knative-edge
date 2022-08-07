package event

type Event struct {
	Type     EventType
	Encoding EncodingType
	Data     string
}

func WrapEvent[T any](etype EventType, event *T) (*Event, error) {
	data, err := Encode(DefaultEncoding, *event)

	if err != nil {
		return nil, err
	}

	return &Event{
		Type: etype,
		Data: string(data),
	}, nil
}
