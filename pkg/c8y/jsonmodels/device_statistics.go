package jsonmodels

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"

// DeviceStatistics contains usage statistics for a single device within a
// given time period.
//
// OAS schema: DeviceStatistics
//
// Example shape:
//
//	{
//	  "count": 42,
//	  "deviceId": "12345",
//	  "deviceParents": ["99999"],   // monthly only
//	  "deviceType": "c8y_Linux"     // monthly only
//	}
type DeviceStatistics struct {
	jsondoc.Facade
}

// NewDeviceStatistics parses raw JSON bytes into a DeviceStatistics value.
func NewDeviceStatistics(b []byte) DeviceStatistics {
	return DeviceStatistics{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// Count returns the sum of measurements, events and alarms created/updated for
// the device in the period.
func (d DeviceStatistics) Count() int64 {
	return d.Get("count").Int()
}

// DeviceID returns the unique identifier of the device.
func (d DeviceStatistics) DeviceID() string {
	return d.Get("deviceId").String()
}

// DeviceParents returns the IDs of parent devices/groups. This field is only
// populated in monthly responses.
func (d DeviceStatistics) DeviceParents() []string {
	raw := d.Get("deviceParents")
	if !raw.Exists() || raw.IsObject() {
		return nil
	}
	var parents []string
	for _, v := range raw.Array() {
		parents = append(parents, v.String())
	}
	return parents
}

// DeviceType returns the value of the `type` field from the corresponding
// managed object. This field is only populated in monthly responses.
func (d DeviceStatistics) DeviceType() string {
	return d.Get("deviceType").String()
}
