package c8y

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// NewMeasurementSourceByName returns the source object by searching for a matching device by name
func (s *MeasurementService) NewMeasurementSourceByName(ctx context.Context, name string) (*MeasurementSource, error) {

	devices, _, err := s.client.Inventory.GetDevicesByName(ctx, name, nil)

	if err != nil {
		return nil, err
	}

	totalFound := len(devices.ManagedObjects)

	if totalFound == 0 {
		return nil, fmt.Errorf("Not matching devices found. The query must return exactly 1 device")
	}

	if totalFound > 1 {
		return nil, fmt.Errorf("More than 1 device was found. The query must return exactly 1 device. Name: %s", name)
	}

	source := &MeasurementSource{
		ID:   devices.ManagedObjects[0].ID,
		Name: devices.ManagedObjects[0].Name,
	}

	return source, nil

}

// MeasurementSource represents a device source.
type MeasurementSource struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// FragmentName is a fragment name which can be added to a measurement object.
// These are commonly used to tag particular measurements with additional information
// In JSON a fragment name will look like this:
// {
//   ...
//   customMarker: {},
//   ...
// }
type FragmentName string

// NewFragmentNameSeries returns a new array of fragment names will can be appended to a particular measurement
func NewFragmentNameSeries(names ...string) []FragmentName {
	fragments := []FragmentName{}

	for _, name := range names {
		fragments = append(fragments, FragmentName(name))
	}
	return fragments
}

// SimpleMeasurementOptions contains the arguments which can be provided when using the NewSimpleMeasurementRepresentation constructor
// Timestamp will be set to time.Now() if it is not provided by the user
type SimpleMeasurementOptions struct {
	SourceID            string
	Timestamp           *time.Time
	Type                string
	ValueFragmentType   string
	ValueFragmentSeries string
	FragmentType        []string
	Value               interface{}
	Unit                string
}

// NewSimpleMeasurementRepresentation returns a measurement with one value Fragment type/series
// It is a helper function to make it easier to create simple measurements that can be added to the platform
func NewSimpleMeasurementRepresentation(opt SimpleMeasurementOptions) (*MeasurementRepresentation, error) {

	if opt.Type == "" {
		return nil, fmt.Errorf("Type must not be an empty string! It is a parameter required by Cumulocity")
	}

	var ts time.Time
	if opt.Timestamp == nil {
		ts = time.Now()
	} else {
		ts = *opt.Timestamp
	}

	m := &MeasurementRepresentation{
		Source: MeasurementSource{
			ID: opt.SourceID,
		},
		Timestamp: ts,
		Type:      opt.Type,
		Fragments: NewFragmentNameSeries(opt.FragmentType...),
		ValueFragmentTypes: []ValueFragmentType{
			ValueFragmentType{
				Name: opt.ValueFragmentType,
				Values: []ValueFragmentSeries{
					ValueFragmentSeries{
						Name:  opt.ValueFragmentSeries,
						Value: opt.Value,
						Unit:  opt.Unit,
					},
				},
			},
		},
	}

	return m, nil
}

// MeasurementRepresentation is the measurement object in order to push it into the Cumulocity platform
type MeasurementRepresentation struct {
	Timestamp          time.Time           `json:"time"`
	Source             MeasurementSource   `json:"source"`
	Type               string              `json:"type"`
	Fragments          []FragmentName      `json:"-"`
	ValueFragmentTypes []ValueFragmentType `json:"-"`
}

// MarshalJSON custom marshalling of the Value Fragment Type representation in a measurement
func (m ValueFragmentType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")

	valueFragmentSeries := []string{}
	for _, value := range m.Values {
		valueStr := ""
		switch v := value.Value.(type) {

		case float64:
			valueStr = fmt.Sprintf("{\"value\": %v, \"unit\":\"%s\" }", v, value.Unit)

		case int64:
			valueStr = fmt.Sprintf("{\"value\": %v, \"unit\":\"%s\" }", v, value.Unit)

		default:
			// Try marshalling it if it is a complex object.
			// This will rely on the user to add marshaling flags to it
			b, err := json.Marshal(value)
			if err != nil {
				return nil, fmt.Errorf("Could not marshal value. %s", err)
			}
			valueStr = string(b)
		}

		valueFragmentSeries = append(valueFragmentSeries, fmt.Sprintf("\"%s\": %s", value.Name, valueStr))
	}

	buffer.WriteString(fmt.Sprintf("\"%s\":{%s}", m.Name, strings.Join(valueFragmentSeries, ",")))

	buffer.WriteString("}")

	return buffer.Bytes(), nil
}

// MarshalJSON converts the Measurement Representation to a json string
// A custom marshalling is required as the meausurement object is structured
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
			return nil, fmt.Errorf("Could not marshal valueFragmentType. %s", err)
		}

		// Remove the "{" and "}" object brackets from the returned result as we need to merge the
		// object properties to the overall measurement.
		parts = append(parts, fmt.Sprintf("%s", b[1:len(b)-1]))
	}

	o := []byte(fmt.Sprintf("{%s}", strings.Join(parts, ",")))
	log.Printf("%s\n", o)

	return o, nil
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
