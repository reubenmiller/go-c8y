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

func (a Alarm) ID() string {
	return a.Get("id").String()
}

func (a Alarm) Type() string {
	return a.Get("type").String()
}

func (a Alarm) Text() string {
	return a.Get("text").String()
}

func (a Alarm) SourceID() string {
	return a.Get("source.id").String()
}

func (a Alarm) Severity() string {
	return a.Get("severity").String()
}

func (a Alarm) Status() string {
	return a.Get("status").String()
}

func (a Alarm) Time() time.Time {
	return a.Get("time").Time()
}

func (a Alarm) CreationTime() time.Time {
	return a.Get("creationTime").Time()
}

func (a Alarm) FirstOccurrenceTime() time.Time {
	return a.Get("firstOccurrenceTime").Time()
}

func (a Alarm) Count() int64 {
	return a.Get("count").Int()
}

func (a Alarm) Self() string {
	return a.Get("self").String()
}
