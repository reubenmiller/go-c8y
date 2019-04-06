package c8y

import (
	"encoding/json"
)

// EventBuilder represents a custom event where the mandatory properies are set via its constructor NewEventBuilder
type EventBuilder struct {
	data map[string]interface{}
}

// NewEventBuilder returns a new custom event with the required fields set.
// The event will have a timestamp set to Now(). The timestamp can be set to another timestamp by using SetTimestamp()
func NewEventBuilder(deviceID string, typeName string, text string) *EventBuilder {
	c := &EventBuilder{
		data: map[string]interface{}{},
	}
	c.SetTimestamp(nil)
	c.SetDeviceID(deviceID)
	c.SetText(text)
	c.SetType(typeName)
	return c
}

// MarshalJSON returns the given event in json format
func (c EventBuilder) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.data)
}

// DeviceID returns the device id of the custom event
func (c EventBuilder) DeviceID() string {
	if v, ok := c.data["source"].(Source); ok {
		return v.ID
	}
	return ""
}

// SetDeviceID sets the device id for the custom event
func (c *EventBuilder) SetDeviceID(ID string) *EventBuilder {
	return c.Set("source", Source{
		ID: ID,
	})
}

// Type returns the event type
func (c EventBuilder) Type() string {
	return c.data["type"].(string)
}

// SetType sets the event type
func (c *EventBuilder) SetType(name string) *EventBuilder {
	return c.Set("type", name)
}

// Text returns the device id of the custom event
func (c EventBuilder) Text() string {
	return c.data["text"].(string)
}

// SetText sets the event text for the custom event
func (c *EventBuilder) SetText(text string) *EventBuilder {
	return c.Set("text", text)
}

// Timestamp returns the timestamp of the custom event
func (c EventBuilder) Timestamp() Timestamp {
	return *c.data["time"].(*Timestamp)
}

// SetTimestamp sets the timestamp when the event was created. If the value is nil, then the current timestamp will be used
func (c *EventBuilder) SetTimestamp(value *Timestamp) *EventBuilder {
	if value == nil {
		value = NewTimestamp()
	}
	return c.Set("time", value)
}

// Set sets the name property with the given value
func (c *EventBuilder) Set(name string, value interface{}) *EventBuilder {
	c.data[name] = value
	return c
}

// Get returns the given property value. If the property does not exist, then the second returne parameter will be set to false
func (c EventBuilder) Get(name string) (interface{}, bool) {
	if v, ok := c.data[name]; ok {
		return v, true
	}
	return nil, false
}
