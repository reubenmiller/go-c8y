package measurements

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
)

// Measurement is the Cumulocity measurement representation in the platform
type Measurement struct {
	ID     string        `json:"id,omitempty"`
	Source *model.Source `json:"source,omitempty"`
	Type   string        `json:"type,omitempty"`
	Self   string        `json:"self,omitempty"`
	Time   time.Time     `json:"time,omitempty,omitzero"`
}

// SimpleMeasurementOptions contains the arguments which can be provided when using the NewSimpleMeasurementRepresentation constructor
// Timestamp will be set to time.Now() if it is not provided by the user
type SimpleMeasurementOptions struct {
	SourceID            string
	Timestamp           time.Time
	Type                string
	ValueFragmentType   string
	ValueFragmentSeries string
	FragmentType        []string
	Value               interface{}
	Unit                string
}

// MeasurementRepresentation is the measurement object in order to push it into the Cumulocity platform
type MeasurementRepresentation struct {
	Timestamp          time.Time           `json:"time"`
	Source             model.Source        `json:"source"`
	Type               string              `json:"type"`
	Fragments          []string            `json:"-"`
	ValueFragmentTypes []ValueFragmentType `json:"-"`
}

// ValueFragmentType represents the Value Fragment Type information
// A Value Fragment Type can have multiple series definitions
// This layout deviates from the Cumulocity Measurement model
type ValueFragmentType struct {
	Name   string
	Values []ValueFragmentSeries
}

// ValueFragmentSeries represents the Value Fragment Series information
// This layout deviates from the Cumulocity Measurement model
type ValueFragmentSeries struct {
	Name  string
	Value interface{}
	Unit  string
}

// NewSimpleMeasurementRepresentation returns a measurement with one value Fragment type/series
// It is a helper function to make it easier to create simple measurements that can be added to the platform
func NewSimpleMeasurementRepresentation(opt SimpleMeasurementOptions) *MeasurementRepresentation {

	if opt.Type == "" {
		opt.Type = "measurement"
	}

	if opt.Timestamp.IsZero() {
		opt.Timestamp = time.Now()
	}

	m := &MeasurementRepresentation{
		Source: model.Source{
			ID: opt.SourceID,
		},
		Timestamp: opt.Timestamp,
		Type:      opt.Type,
		Fragments: opt.FragmentType,
		ValueFragmentTypes: []ValueFragmentType{
			{
				Name: opt.ValueFragmentType,
				Values: []ValueFragmentSeries{
					{
						Name:  opt.ValueFragmentSeries,
						Value: opt.Value,
						Unit:  opt.Unit,
					},
				},
			},
		},
	}

	return m
}

var ErrInvalidMeasurementValueType = errors.New("only numbers are supported as measurement values")

// MarshalJSON custom marshaling of the Value Fragment Type representation in a measurement
func (m ValueFragmentType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")

	setMeasurementValue := func(val interface{}, unit string) string {
		return fmt.Sprintf("{\"value\": %v, \"unit\":\"%s\" }", val, unit)
	}

	valueFragmentSeries := []string{}
	for _, value := range m.Values {
		valueStr := ""
		switch v := value.Value.(type) {

		case []byte:
			return nil, ErrInvalidMeasurementValueType
		case []rune:
			return nil, ErrInvalidMeasurementValueType
		case string:
			return nil, ErrInvalidMeasurementValueType
		case bool:
			// convert boolean to an integer
			if v {
				valueStr = setMeasurementValue(1, value.Unit)
			} else {
				valueStr = setMeasurementValue(0, value.Unit)
			}
		case int:
			valueStr = setMeasurementValue(v, value.Unit)
		case int8:
			valueStr = setMeasurementValue(v, value.Unit)
		case int16:
			valueStr = setMeasurementValue(v, value.Unit)
		case int32:
			valueStr = setMeasurementValue(v, value.Unit)
		case int64:
			valueStr = setMeasurementValue(v, value.Unit)
		case uint:
			valueStr = setMeasurementValue(v, value.Unit)
		case uint8:
			valueStr = setMeasurementValue(v, value.Unit)
		case uint16:
			valueStr = setMeasurementValue(v, value.Unit)
		case uint32:
			valueStr = setMeasurementValue(v, value.Unit)
		case uint64:
			valueStr = setMeasurementValue(v, value.Unit)
		case float32:
			valueStr = setMeasurementValue(v, value.Unit)
		case float64:
			valueStr = setMeasurementValue(v, value.Unit)

		default:
			return nil, ErrInvalidMeasurementValueType
		}

		valueFragmentSeries = append(valueFragmentSeries, fmt.Sprintf("\"%s\": %s", value.Name, valueStr))
	}

	buffer.WriteString(fmt.Sprintf("\"%s\":{%s}", m.Name, strings.Join(valueFragmentSeries, ",")))

	buffer.WriteString("}")

	return buffer.Bytes(), nil
}

// MarshalJSON converts the Measurement Representation to a json string
// A custom marshaling is required as the measurement object is structured
// differently to the official Cumulocity Measurement structure to make it easier to handle
func (m MeasurementRepresentation) MarshalJSON() ([]byte, error) {
	// Collect the json property strings, then join all of the parts together at the end
	parts := []string{}

	// Source
	parts = append(parts, fmt.Sprintf("\"source\":{\"id\":\"%s\"}", m.Source.ID))

	// Type
	if m.Type != "" {
		parts = append(parts, fmt.Sprintf("\"type\":\"%s\"", m.Type))
	}

	// Timestamp
	parts = append(parts, fmt.Sprintf("\"time\":\"%s\"", m.Timestamp.Format(time.RFC3339)))

	// Custom Fragments
	for _, fragmentName := range m.Fragments {
		parts = append(parts, fmt.Sprintf("\"%s\":{}", fragmentName))
	}

	// Values
	for _, valueFragmentType := range m.ValueFragmentTypes {

		b, err := json.Marshal(valueFragmentType)
		if err != nil {
			return nil, fmt.Errorf("could not marshal valueFragmentType. %s", err)
		}

		// Remove the "{" and "}" object brackets from the returned result as we need to merge the
		// object properties to the overall measurement.
		parts = append(parts, string(b[1:len(b)-1]))
	}

	o := []byte(fmt.Sprintf("{%s}", strings.Join(parts, ",")))
	slog.Info(string(o))

	return o, nil
}
