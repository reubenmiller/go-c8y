package c8y_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func TestAlarmBuilder_CreateAlarmBuilder(t *testing.T) {

	id := "12345"
	alarmType := "testType"
	alarmText := "Custom Alarm 1"
	timestamp, err := time.Parse(time.RFC3339, "2019-04-06T14:11:42.045421+02:00")
	testingutils.Ok(t, err)
	wantJSON := `{"severity":"MAJOR","source":{"id":"12345"},"text":"Custom Alarm 1","time":"2019-04-06T14:11:42.045421+02:00","type":"testType"}`

	builder := c8y.NewAlarmBuilder(id, alarmType, alarmText).SetTimestamp(c8y.NewTimestamp(timestamp))

	alarmJSON, err := builder.MarshalJSON()
	testingutils.Ok(t, err)
	testingutils.Equals(t, wantJSON, string(alarmJSON))
}

func TestAlarmBuilder_CreateAlarm(t *testing.T) {
	client := createTestClient()
	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)
	builder := c8y.NewAlarmBuilder(testDevice.ID, "testType", "Custom Event 1")

	alarmJSON, err := builder.MarshalJSON()
	testingutils.Ok(t, err)
	testingutils.Assert(t, string(alarmJSON) != "", "alarm json should be valid")

	alarm, resp, err := client.Alarm.Create(
		context.Background(),
		builder,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Equals(t, testDevice.ID, alarm.Source.ID)
	testingutils.Equals(t, "testType", alarm.Type)
	testingutils.Equals(t, "Custom Event 1", alarm.Text)
}

func TestAlarmBuilder_Timestamp(t *testing.T) {
	timestampStr := "2019-04-06T14:11:42.045421+02:00"
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	testingutils.Ok(t, err)

	builder := c8y.NewAlarmBuilder("12345", "customAlarm1", "Custom Event Text 1")

	builder.SetTimestamp(c8y.NewTimestamp(timestamp))
	eventTimestamp := builder.Timestamp()
	testingutils.Equals(t, timestampStr, eventTimestamp.String())
}

func TestAlarmBuilder_DeviceID(t *testing.T) {
	builder := c8y.NewAlarmBuilder("12345", "customAlarm1", "Custom Event Text 1")
	testingutils.Equals(t, "12345", builder.DeviceID())

	builder.SetDeviceID("99192")
	testingutils.Equals(t, "99192", builder.DeviceID())
}

func TestAlarmBuilder_Type(t *testing.T) {
	builder := c8y.NewAlarmBuilder("12345", "customAlarm1", "Custom Event Text 1")
	testingutils.Equals(t, "customAlarm1", builder.Type())

	builder.SetType("anotherType")
	testingutils.Equals(t, "anotherType", builder.Type())
}

func TestAlarmBuilder_Text(t *testing.T) {
	builder := c8y.NewAlarmBuilder("12345", "customAlarm1", "Custom Event Text 1")
	testingutils.Equals(t, "Custom Event Text 1", builder.Text())

	builder.SetText("Some other Alarm Text")
	testingutils.Equals(t, "Some other Alarm Text", builder.Text())
}

func TestAlarmBuilder_Severity(t *testing.T) {
	builder := c8y.NewAlarmBuilder("12345", "customAlarm1", "Custom Event Text 1")
	// Defaults to Major
	testingutils.Equals(t, c8y.AlarmSeverityMajor, builder.Severity())

	builder.SetSeverityCritical()
	testingutils.Equals(t, c8y.AlarmSeverityCritical, builder.Severity())

	builder.SetSeverityMajor()
	testingutils.Equals(t, c8y.AlarmSeverityMajor, builder.Severity())

	builder.SetSeverityMinor()
	testingutils.Equals(t, c8y.AlarmSeverityMinor, builder.Severity())

	builder.SetSeverityWarning()
	testingutils.Equals(t, c8y.AlarmSeverityWarning, builder.Severity())
}

func TestAlarmBuilder_GetSet(t *testing.T) {
	builder := c8y.NewAlarmBuilder("12345", "customAlarm1", "Custom Event Text 1")

	builder.Set("c8y_CustomFragment", int64(2))
	val, ok := builder.Get("c8y_CustomFragment")
	testingutils.Equals(t, true, ok)
	testingutils.Equals(t, int64(2), val.(int64))

	_, ok = builder.Get("c8y_NonExistentProp")
	testingutils.Equals(t, false, ok)
}
