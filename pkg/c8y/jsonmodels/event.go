package jsonmodels

import (
	"encoding/json"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
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

// SourceID returns the id of the managed object the event is associated with.
// Hand-written: the OAS models `source` as a nested object, so this accessor is not
// mechanically derivable. The scalar accessors (ID, Type, Time, ...) are generated in
// zz_generated_event.go.
func (e Event) SourceID() string {
	return e.Get("source.id").String()
}
