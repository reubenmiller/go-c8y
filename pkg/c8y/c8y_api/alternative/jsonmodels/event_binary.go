package jsonmodels

import "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"

type EventBinary struct {
	jsondoc.JSONDoc
}

func NewEventBinary(b []byte) EventBinary {
	return EventBinary{jsondoc.New(b)}
}

// Self returns the URL to this resource
func (e EventBinary) Self() string {
	return e.Get("self").String()
}

// Type returns the media type of the attachment
func (e EventBinary) Type() string {
	return e.Get("type").String()
}

// Source returns the unique identifier of the event
func (e EventBinary) Source() string {
	return e.Get("source").String()
}

// Name returns the name of the attachment
func (e EventBinary) Name() string {
	return e.Get("name").String()
}
