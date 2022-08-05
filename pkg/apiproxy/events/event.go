package events

type Event struct {
	Type     string
	Encoding EncodingType
	Data     string
}
