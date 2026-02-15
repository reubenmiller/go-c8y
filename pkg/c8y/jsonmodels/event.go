package jsonmodels

import (
	"encoding/json"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
)

type Event struct {
	jsondoc.Facade
}

func NewEvent(b []byte) Event {
	return Event{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func NewEventWithType(sourceID, eventType, text string, fragments map[string]any) Event {
	data := map[string]any{
		"source": map[string]any{
			"id": sourceID,
		},
		"type": eventType,
		"text": text,
		"time": time.Now(),
	}
	for k, v := range fragments {
		data[k] = v
	}
	b, _ := json.Marshal(data)
	return Event{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (e Event) ID() string {
	return e.Get("id").String()
}

func (e Event) Type() string {
	return e.Get("type").String()
}

func (e Event) Text() string {
	return e.Get("text").String()
}

func (e Event) SourceID() string {
	return e.Get("source.id").String()
}

func (e Event) Time() time.Time {
	return e.Get("time").Time()
}

func (e Event) Self() string {
	return e.Get("self").String()
}

func (e Event) CreationTime() time.Time {
	return e.Get("creationTime").Time()
}

func (e Event) LastUpdated() time.Time {
	return e.Get("lastUpdated").Time()
}
