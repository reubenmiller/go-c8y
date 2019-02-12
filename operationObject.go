package c8y

import "github.com/tidwall/gjson"

// Operation todo
type Operation struct {
	ID           string     `json:"id,omitempty"`
	CreationTime *Timestamp `json:"creationTime,omitempty"`
	DeviceID     string     `json:"deviceId,omitempty"`
	Type         string     `json:"type,omitempty"`
	DeviceName   string     `json:"deviceName,omitempty"`
	Status       string     `json:"status,omitempty"`
	Description  string     `json:"description,omitempty"`
	Self         string     `json:"self,omitempty"`
	EventID      string     `json:"eventId,omitempty"`

	Item gjson.Result `json:"-"`
}
