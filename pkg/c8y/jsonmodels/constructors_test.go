package jsonmodels

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The constructors which build a document from values set the fields they
// document, and leave the optional ones out when empty.
func TestConstructors_BuildDocuments(t *testing.T) {
	t.Run("alarm", func(t *testing.T) {
		a := NewAlarmWithType("12", "c8y_Test", "MAJOR", "text", map[string]any{"custom": 1})
		assert.Equal(t, "12", a.Get("source.id").String())
		assert.Equal(t, "c8y_Test", a.Get("type").String())
		assert.Equal(t, "MAJOR", a.Get("severity").String())
		assert.Equal(t, "text", a.Get("text").String())
		assert.True(t, a.Get("time").Exists())
		assert.Equal(t, int64(1), a.Get("custom").Int())
	})
	t.Run("event", func(t *testing.T) {
		e := NewEventWithType("12", "c8y_Test", "text", map[string]any{"custom": true})
		assert.Equal(t, "12", e.Get("source.id").String())
		assert.Equal(t, "c8y_Test", e.Get("type").String())
		assert.Equal(t, "text", e.Get("text").String())
		assert.True(t, e.Get("custom").Bool())
	})
	t.Run("measurement", func(t *testing.T) {
		m := NewMeasurementWithType("12", "c8y_Temp", map[string]any{"c8y_Temp": map[string]any{"T": map[string]any{"value": 1.5}}})
		assert.Equal(t, "12", m.Get("source.id").String())
		assert.Equal(t, "c8y_Temp", m.Get("type").String())
		assert.Equal(t, 1.5, m.Get("c8y_Temp.T.value").Float())
	})
	t.Run("operation", func(t *testing.T) {
		o := NewOperationWithStatus("12", "PENDING", map[string]any{"c8y_Restart": map[string]any{}})
		assert.Equal(t, "12", o.Get("deviceId").String())
		assert.Equal(t, "PENDING", o.Get("status").String())
		assert.True(t, o.Get("c8y_Restart").Exists())
	})
	t.Run("device with type", func(t *testing.T) {
		d := NewDeviceWithType("name", "c8y_Test")
		assert.Equal(t, "name", d.Get("name").String())
		assert.Equal(t, "c8y_Test", d.Get("type").String())
		assert.True(t, d.Get("c8y_IsDevice").Exists())
	})
	t.Run("repository items", func(t *testing.T) {
		cfg := NewConfigurationWithOptions("cfg", "tedge.toml")
		assert.Equal(t, "c8y_ConfigurationDump", cfg.Get("type").String())
		assert.Equal(t, "cfg", cfg.Get("name").String())
		assert.Equal(t, "tedge.toml", cfg.Get("configurationType").String())
		assert.False(t, NewConfigurationWithOptions("", "").Get("name").Exists())

		fw := NewFirmwareWithOptions("fw")
		assert.Equal(t, "c8y_Firmware", fw.Get("type").String())
		assert.Equal(t, "fw", fw.Get("name").String())

		fwv := NewFirmwareVersionWithOptions("1.0.0", "https://example.com/fw")
		assert.Equal(t, "c8y_FirmwareBinary", fwv.Get("type").String())
		assert.Equal(t, "1.0.0", fwv.Get("c8y_Firmware.version").String())
		assert.Equal(t, "https://example.com/fw", fwv.Get("c8y_Firmware.url").String())
		assert.False(t, NewFirmwareVersionWithOptions("1.0.0", "").Get("c8y_Firmware.url").Exists())

		sw := NewSoftwareWithOptions("sw", "apt")
		assert.Equal(t, "c8y_Software", sw.Get("type").String())
		assert.Equal(t, "sw", sw.Get("name").String())
		assert.Equal(t, "apt", sw.Get("softwareType").String())

		swv := NewSoftwareVersionWithOptions("2.0.0", "https://example.com/sw")
		assert.Equal(t, "2.0.0", swv.Get("c8y_Software.version").String())
		assert.Equal(t, "https://example.com/sw", swv.Get("c8y_Software.url").String())
	})
}

// The constructors which wrap JSON keep the document readable through the accessors.
func TestConstructors_WrapDocuments(t *testing.T) {
	setting := NewMicroserviceSetting([]byte(`{"key":"k","defaultValue":"v"}`))
	assert.Equal(t, "k", setting.Key())
	assert.Equal(t, "v", setting.DefaultValue())

	creds := NewRemoteAccessCredentials([]byte(`{"type":"USER_PASS","username":"u"}`))
	assert.Equal(t, "USER_PASS", creds.Type())
	assert.Equal(t, "u", creds.Username())

	series := NewMeasurementSeriesWithDevice([]byte(`{}`), "12", "device")
	assert.Equal(t, "12", series.DeviceID)
	assert.Equal(t, "device", series.DeviceName)
}
