package c8y

import (
	"encoding/json"
)

// AlarmBuilder represents a alarm where the mandatory properies are set via its constructor NewAlarmBuilder
type AlarmBuilder struct {
	data map[string]interface{}
}

// NewAlarmBuilder returns a new alarm builder with the required fields set.
// The alarm will have a timestamp set to Now(). The timestamp can be set to another timestamp by using SetTimestamp()
// The alarm will have a default severity of MAJOR, but it can be changed by using
// .SetSeverity functions
func NewAlarmBuilder(deviceID string, typeName string, text string) *AlarmBuilder {
	c := &AlarmBuilder{
		data: map[string]interface{}{},
	}
	c.SetTimestamp(nil)
	c.SetType(typeName)
	c.SetText(text)
	c.SetDeviceID(deviceID)
	c.SetSeverityMajor() // Set default severity to MAJOR
	return c
}

// MarshalJSON returns the given event in json format
func (c AlarmBuilder) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.data)
}

// SetSeverityMajor sets the alarm serverity to Major
func (c *AlarmBuilder) SetSeverityMajor() *AlarmBuilder {
	return c.Set("severity", AlarmSeverityMajor)
}

// SetSeverityMinor sets the alarm serverity to Minor
func (c *AlarmBuilder) SetSeverityMinor() *AlarmBuilder {
	return c.Set("severity", AlarmSeverityMinor)
}

// SetSeverityCritical sets the alarm serverity to Critical
func (c *AlarmBuilder) SetSeverityCritical() *AlarmBuilder {
	return c.Set("severity", AlarmSeverityCritical)
}

// SetSeverityWarning sets the alarm serverity to Warning
func (c *AlarmBuilder) SetSeverityWarning() *AlarmBuilder {
	return c.Set("severity", AlarmSeverityWarning)
}

// DeviceID returns the device id of the alarm
func (c AlarmBuilder) DeviceID() string {
	if v, ok := c.data["source"].(Source); ok {
		return v.ID
	}
	return ""
}

// SetDeviceID sets the device id for the alarm
func (c *AlarmBuilder) SetDeviceID(ID string) *AlarmBuilder {
	return c.Set("source", Source{
		ID: ID,
	})
}

// Text returns the device id of the alarm
func (c AlarmBuilder) Text() string {
	return c.data["text"].(string)
}

// SetText sets the alarm text
func (c *AlarmBuilder) SetText(ID string) *AlarmBuilder {
	return c.Set("text", ID)
}

// Type returns the alarm type
func (c AlarmBuilder) Type() string {
	return c.data["type"].(string)
}

// SetType sets the alarm type
func (c *AlarmBuilder) SetType(ID string) *AlarmBuilder {
	return c.Set("type", ID)
}

// Timestamp returns the timestamp of the alarm
func (c AlarmBuilder) Timestamp() Timestamp {
	return *c.data["time"].(*Timestamp)
}

// SetTimestamp sets the timestamp when the event was created. If the value is nil, then the current timestamp will be used
func (c *AlarmBuilder) SetTimestamp(value *Timestamp) *AlarmBuilder {
	if value == nil {
		value = NewTimestamp()
	}
	return c.Set("time", value)
}

// Set sets the name property with the given value
func (c *AlarmBuilder) Set(name string, value interface{}) *AlarmBuilder {
	c.data[name] = value
	return c
}

// Get returns the given property value. If the property does not exist, then the second parameter will be set to false
func (c AlarmBuilder) Get(name string) (interface{}, bool) {
	if v, ok := c.data[name]; ok {
		return v, true
	}
	return nil, false
}
