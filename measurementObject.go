package c8y

import "github.com/tidwall/gjson"

// MeasurementObject is the Cumulocity measurement representation in the platform
type MeasurementObject struct {
	ID     string     `json:"id,omitempty"`
	Source *Source    `json:"source,omitempty"`
	Type   string     `json:"type,omitempty"`
	Self   string     `json:"self,omitempty"`
	Time   *Timestamp `json:"time,omitempty"`

	Item gjson.Result `json:"-"`
}
