package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// SupportedMeasurements is the list of measurement types supported by a managed object
type SupportedMeasurements struct {
	Values []string `json:"c8y_SupportedMeasurements,omitempty"`
}

// SupportedSeries is the list of measurement series in the form of "<fragment>.<series>"
// that are supported by a managed object
type SupportedSeries struct {
	Values []string `json:"c8y_SupportedSeries,omitempty"`
}

// MeasurementRepresentationCollection is the generic data structure which contains the response cumulocity when requesting a measurement collection
type MeasurementRepresentationCollection struct {
	*BaseResponse

	Measurements []MeasurementRepresentation `json:"measurements"`
}

type MeasurementCollection struct {
	*BaseResponse

	Measurements []Measurement `json:"measurements,omitempty"`
}

// MeasurementSeriesDefinition represents information about a single series
type MeasurementSeriesDefinition struct {
	Unit string `json:"unit"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// MeasurementSeriesValueGroup represents multiple values for multiple series for a single timestamp
type MeasurementSeriesValueGroup struct {
	Timestamp time.Time `json:"timestamp,omitempty,omitzero"`
	Values    []float64 `json:"values"`
}

// MeasurementSeriesAggregateValueGroup represents multiple aggregate values for multiple series for a single timestamp
type MeasurementSeriesAggregateValueGroup struct {
	Timestamp time.Time                   `json:"timestamp"`
	Values    []MeasurementAggregateValue `json:"values"`
}

// MeasurementSeriesGroup represents a group of series values (no aggregate values)
type MeasurementSeriesGroup struct {
	DeviceID  string                        `json:"deviceId"`
	Series    []MeasurementSeriesDefinition `json:"series"`
	Values    []MeasurementSeriesValueGroup `json:"values"`
	DateFrom  time.Time                     `json:"dateFrom"`
	DateTo    time.Time                     `json:"dateTo"`
	Truncated bool                          `json:"truncated"`
}

// MeasurementSeriesAggregateGroup represents a group of aggregate series
type MeasurementSeriesAggregateGroup struct {
	Series    []MeasurementSeriesDefinition          `json:"series"`
	Values    []MeasurementSeriesAggregateValueGroup `json:"values"`
	DateFrom  time.Time                              `json:"dateFrom"`
	DateTo    time.Time                              `json:"dateTo"`
	Truncated bool                                   `json:"truncated"`
}

// MeasurementAggregateValue represents the aggregate value of a single measurement.
type MeasurementAggregateValue struct {
	Min Number `json:"min"`
	Max Number `json:"max"`
}

// UnmarshalJSON converts the Cumulocity measurement Series response to a format which is easier to parse.
//
//	{
//	    "series": [ "c8y_Temperature.A", "c8y_Temperature.B" ],
//	    "unit": [ "degC", "degC" ],
//	    "truncated": true,
//	    "values": [
//	        { "timestamp": "2018-11-11T23:20:00.000+01:00", values: [0.0001, 0.1001] },
//	        { "timestamp": "2018-11-11T23:20:01.000+01:00", values: [0.1234, 2.2919] },
//	        { "timestamp": "2018-11-11T23:20:02.000+01:00", values: [0.8370, 4.8756] }
//	    ]
//	}
func (d *MeasurementSeriesGroup) UnmarshalJSON(data []byte) error {
	c8ySeries := gjson.ParseBytes(data)

	// Get the series definitions
	var seriesDefinitions []MeasurementSeriesDefinition

	c8ySeries.Get("series").ForEach(func(_, item gjson.Result) bool {
		v := &MeasurementSeriesDefinition{}
		if err := json.Unmarshal([]byte(item.String()), &v); err != nil {
			slog.Info("Could not unmarshal series definition", "value", item.String())
		}

		seriesDefinitions = append(seriesDefinitions, *v)
		return true
	})

	d.Series = seriesDefinitions
	d.Truncated = c8ySeries.Get("truncated").Bool()

	totalSeries := len(seriesDefinitions)

	// Get each series values
	var allSeries []MeasurementSeriesValueGroup
	c8ySeries.Get("values").ForEach(func(key, values gjson.Result) bool {
		timestamp, err := time.Parse(time.RFC3339, key.Str)

		if err != nil {
			panic(fmt.Sprintf("Invalid timestamp: %s", key.Str))
		}

		seriesValues := &MeasurementSeriesValueGroup{
			Timestamp: timestamp,
			Values:    make([]float64, totalSeries),
		}

		index := 0
		values.ForEach(func(_, value gjson.Result) bool {
			// Note: min and max values are the same when no aggregation is being used!
			// so technically we could get the value from either min or max.
			seriesValues.Values[index] = value.Get("max").Float()
			index++
			return true
		})

		allSeries = append(allSeries, *seriesValues)
		return true
	})

	// Store the first and last timestamps
	if len(allSeries) > 0 {
		d.DateFrom = allSeries[0].Timestamp
		d.DateTo = allSeries[len(allSeries)-1].Timestamp
	}

	d.Values = allSeries
	return nil
}

// UnmarshalJSON controls the conversion of json bytes to the MeasurementSeriesAggregateGroup struct
func (d *MeasurementSeriesAggregateGroup) UnmarshalJSON(data []byte) error {
	c8ySeries := gjson.ParseBytes(data)

	// Get the series definitions
	var seriesDefinitions []MeasurementSeriesDefinition

	c8ySeries.Get("series").ForEach(func(_, item gjson.Result) bool {
		v := &MeasurementSeriesDefinition{}
		if err := json.Unmarshal([]byte(item.String()), &v); err != nil {
			slog.Info("Could not unmarshal series definition", "value", item.String())
		}

		seriesDefinitions = append(seriesDefinitions, *v)
		return true
	})

	d.Series = seriesDefinitions
	d.Truncated = c8ySeries.Get("truncated").Bool()

	totalSeries := len(seriesDefinitions)

	// Get each series values
	var allSeries []MeasurementSeriesAggregateValueGroup
	c8ySeries.Get("values").ForEach(func(key, values gjson.Result) bool {

		slog.Info("Series", "key", key, "values", values)

		timestamp, err := time.Parse(time.RFC3339, key.Str)

		if err != nil {
			panic(fmt.Sprintf("Invalid timestamp: %s", key.Str))
		}

		seriesValues := &MeasurementSeriesAggregateValueGroup{
			Timestamp: timestamp,
			Values:    make([]MeasurementAggregateValue, totalSeries),
		}

		index := 0
		values.ForEach(func(_, value gjson.Result) bool {
			slog.Info("Current value", "value", value)
			v := &MeasurementAggregateValue{}
			json.Unmarshal([]byte(value.String()), &v)

			slog.Info("Full Value", "value", v)

			seriesValues.Values[index] = *v
			index++
			return true
		})

		allSeries = append(allSeries, *seriesValues)
		return true
	})

	// Store the first and last timestamps
	if len(allSeries) > 0 {
		d.DateFrom = allSeries[0].Timestamp
		d.DateTo = allSeries[len(allSeries)-1].Timestamp
	}

	d.Values = allSeries
	return nil
}

// MarshalCSV converts the measurement series group to a csv output so it can be more easily parsed by other languages
// Example output
// timestamp,c8y_Temperature.A,c8y_Temperature.B
// 2018-11-23T00:45:39+01:00,60.699993,44.300003
// 2018-11-23T01:45:39+01:00,67.63333,47.199997
func (d *MeasurementSeriesGroup) MarshalCSV(delimiter string) ([]byte, error) {

	useDelimiter := delimiter
	if useDelimiter == "" {
		useDelimiter = ","
	}

	totalSeries := len(d.Series)

	// First column is the timestamp
	headers := make([]string, totalSeries+1)
	row := make([]string, totalSeries+1)

	headers[0] = "timestamp"

	var output string

	for i, header := range d.Series {
		headers[i+1] = fmt.Sprintf("%s.%s", header.Type, header.Name)
		output = strings.Join(headers, useDelimiter) + "\n"
	}

	for _, datapoint := range d.Values {
		row[0] = datapoint.Timestamp.Format(time.RFC3339)
		for i := 0; i < totalSeries; i++ {
			row[i+1] = fmt.Sprintf("%f", datapoint.Values[i])
		}
		output += strings.Join(row, useDelimiter) + "\n"
	}

	return []byte(output), nil
}

// Measurement is the Cumulocity measurement representation in the platform
type Measurement struct {
	ID     string    `json:"id,omitempty"`
	Source *Source   `json:"source,omitempty"`
	Type   string    `json:"type,omitempty"`
	Self   string    `json:"self,omitempty"`
	Time   time.Time `json:"time,omitempty,omitzero"`
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
	Value               any
	Unit                string
}

// MeasurementRepresentation is the measurement object in order to push it into the Cumulocity platform
type MeasurementRepresentation struct {
	Timestamp          time.Time           `json:"time,omitempty,omitzero"`
	Source             Source              `json:"source"`
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
	Value any
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
		Source: Source{
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

	setMeasurementValue := func(val any, unit string) string {
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
