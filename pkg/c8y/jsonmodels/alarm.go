package jsonmodels

import (
	"encoding/json"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

type Alarm struct {
	jsondoc.Facade
}

func NewAlarm(b []byte) Alarm {
	return Alarm{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func NewAlarmWithType(sourceID, alarmType, severity, text string, fragments map[string]any) Alarm {
	data := map[string]any{
		"source": map[string]any{
			"id": sourceID,
		},
		"type":     alarmType,
		"severity": severity,
		"text":     text,
		"time":     time.Now(),
	}
	for k, v := range fragments {
		data[k] = v
	}
	b, _ := json.Marshal(data)
	return Alarm{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// SourceID returns the id of the managed object the alarm is associated with.
// Hand-written: the OAS models `source` as a nested object, so this accessor is not
// mechanically derivable. The scalar accessors (ID, Type, Severity, ...) are generated
// in zz_generated_alarm.go.
func (a Alarm) SourceID() string {
	return a.Get("source.id").String()
}
