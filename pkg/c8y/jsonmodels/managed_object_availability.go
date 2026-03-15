package jsonmodels

import (
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

type ManagedObjectAvailability struct {
	jsondoc.Facade
}

func NewManagedObjectAvailability(b []byte) ManagedObjectAvailability {
	return ManagedObjectAvailability{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// DeviceID returns the identifier of the monitored device.
func (m ManagedObjectAvailability) DeviceID() string {
	return m.Get("deviceId").String()
}

// ExternalID returns the identifier used in the external system.
func (m ManagedObjectAvailability) ExternalID() string {
	return m.Get("externalId").String()
}

// LastMessage returns the time when the device last sent a message to Cumulocity.
func (m ManagedObjectAvailability) LastMessage() time.Time {
	return m.Get("lastMessage").Time()
}

// Interval returns the required monitoring interval (in minutes) for the device.
func (m ManagedObjectAvailability) Interval() int64 {
	return m.Get("interval").Int()
}

// DataStatus returns the data availability status (e.g. "AVAILABLE", "UNAVAILABLE").
func (m ManagedObjectAvailability) DataStatus() string {
	return m.Get("dataStatus").String()
}

// ConnectionStatus returns the connection status (e.g. "CONNECTED", "DISCONNECTED").
func (m ManagedObjectAvailability) ConnectionStatus() string {
	return m.Get("connectionStatus").String()
}
