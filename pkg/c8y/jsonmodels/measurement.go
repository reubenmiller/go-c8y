package jsonmodels

import (
	"encoding/json"
	"iter"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

type Measurement struct {
	jsondoc.Facade
}

func NewMeasurement(b []byte) Measurement {
	return Measurement{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func NewMeasurementWithType(sourceID, measurementType string, fragments map[string]any) Measurement {
	data := map[string]any{
		"source": map[string]any{
			"id": sourceID,
		},
		"type": measurementType,
		"time": time.Now(),
	}
	for k, v := range fragments {
		data[k] = v
	}
	b, _ := json.Marshal(data)
	return Measurement{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (m Measurement) ID() string {
	return m.Get("id").String()
}

func (m Measurement) Type() string {
	return m.Get("type").String()
}

func (m Measurement) SourceID() string {
	return m.Get("source.id").String()
}

func (m Measurement) Time() time.Time {
	return m.Get("time").Time()
}

func (m Measurement) Self() string {
	return m.Get("self").String()
}

// IterAs returns an iterator over measurements in the collection.
// This properly constructs Measurement instances from the underlying JSON data.
func (m Measurement) IterAs() iter.Seq[Measurement] {
	return jsondoc.IterWith(m.Iter(), NewMeasurement)
}
