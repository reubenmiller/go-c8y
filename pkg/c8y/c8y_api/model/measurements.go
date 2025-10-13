package model

// SupportedMeasurements is the list of measurement types supported by a managed object
type SupportedMeasurements struct {
	Values []string `json:"c8y_SupportedMeasurements,omitempty"`
}

// SupportedSeries is the list of measurement series in the form of "<fragment>.<series>"
// that are supported by a managed object
type SupportedSeries struct {
	Values []string `json:"c8y_SupportedSeries,omitempty"`
}
