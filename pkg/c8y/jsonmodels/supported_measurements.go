package jsonmodels

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"

type SupportedMeasurements struct {
	jsondoc.JSONDoc
}

func NewSupportedMeasurements(b []byte) SupportedMeasurements {
	return SupportedMeasurements{jsondoc.New(b)}
}

// Values returns the list of supported measurement types
func (s SupportedMeasurements) Values() []string {
	result := s.Get("c8y_SupportedMeasurements")
	if !result.Exists() {
		return []string{}
	}
	values := []string{}
	for _, v := range result.Array() {
		values = append(values, v.String())
	}
	return values
}

type SupportedSeries struct {
	jsondoc.JSONDoc
}

func NewSupportedSeries(b []byte) SupportedSeries {
	return SupportedSeries{jsondoc.New(b)}
}

// Values returns the list of supported measurement series in the form of "<fragment>.<series>"
func (s SupportedSeries) Values() []string {
	result := s.Get("c8y_SupportedSeries")
	if !result.Exists() {
		return []string{}
	}
	values := []string{}
	for _, v := range result.Array() {
		values = append(values, v.String())
	}
	return values
}
