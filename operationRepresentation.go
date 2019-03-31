package c8y

import (
	"encoding/json"
	"errors"

	"github.com/tidwall/gjson"
)

// Operation todo
type Operation struct {
	ID            string     `json:"id,omitempty"`
	CreationTime  *Timestamp `json:"creationTime,omitempty"`
	DeviceID      string     `json:"deviceId,omitempty"`
	DeviceName    string     `json:"deviceName,omitempty"`
	Status        string     `json:"status,omitempty"`
	Description   string     `json:"description,omitempty"`
	Self          string     `json:"self,omitempty"`
	EventID       string     `json:"eventId,omitempty"`
	FailureReason string     `json:"failureReason,omitempty"`

	Item gjson.Result `json:"-"`
}

// CustomOperation is a generic operation representation that can be used to build custom operations with a free format
type CustomOperation struct {
	data map[string]interface{}
}

// NewCustomOperation returns a new Custom Operation with the specified device id
func NewCustomOperation(deviceID string) *CustomOperation {
	op := &CustomOperation{}
	op.data = make(map[string]interface{})
	op.data["deviceId"] = deviceID
	return op
}

// MarshalJSON returns the given operation in json format
func (o CustomOperation) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.data)
}

// DeviceID returns the device id of the operation
func (o CustomOperation) DeviceID() string {
	return o.data["deviceId"].(string)
}

// Set sets the name property with the given value
func (o *CustomOperation) Set(name string, value interface{}) *CustomOperation {
	o.data[name] = value
	return o
}

// Get returns the given property value. If the property does not exist, then the error will have a non nil value
func (o CustomOperation) Get(name string) (interface{}, error) {
	if v, ok := o.data[name]; ok {
		return v, nil
	}
	return nil, errors.New("Unknown property")
}
