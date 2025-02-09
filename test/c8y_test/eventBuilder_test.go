package c8y_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func TestEventBuilder_CreateEventBuilder(t *testing.T) {

	id := "12345"
	eventType := "testType"
	eventText := "Custom Event 1"
	timestamp, err := time.Parse(time.RFC3339, "2019-04-06T14:11:42.045421+02:00")
	testingutils.Ok(t, err)
	wantJSON := `{"source":{"id":"12345"},"text":"Custom Event 1","time":"2019-04-06T14:11:42.045421+02:00","type":"testType"}`

	builder := c8y.NewEventBuilder(id, eventType, eventText).SetTimestamp(c8y.NewTimestamp(timestamp))

	eventJSON, err := builder.MarshalJSON()
	testingutils.Ok(t, err)
	testingutils.Equals(t, wantJSON, string(eventJSON))
}

func TestEventBuilder_CreateEvent(t *testing.T) {
	client := createTestClient()
	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)
	builder := c8y.NewEventBuilder(testDevice.ID, "testType", "Custom Event 1")

	eventJson, err := builder.MarshalJSON()
	testingutils.Ok(t, err)
	testingutils.Assert(t, string(eventJson) != "", "event json should be valid")

	event, resp, err := client.Event.Create(
		context.Background(),
		builder,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Equals(t, testDevice.ID, event.Source.ID)
	testingutils.Equals(t, "testType", event.Type)
	testingutils.Equals(t, "Custom Event 1", event.Text)
}

func TestEventBuilder_Timestamp(t *testing.T) {
	timestampStr := "2019-04-06T14:11:42.045421+02:00"
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	testingutils.Ok(t, err)

	builder := c8y.NewEventBuilder("12345", "customEvent1", "Custom Event Text 1")

	builder.SetTimestamp(c8y.NewTimestamp(timestamp))
	eventTimestamp := builder.Timestamp()
	testingutils.Equals(t, timestampStr, eventTimestamp.String())
}

func TestEventBuilder_DeviceID(t *testing.T) {
	builder := c8y.NewEventBuilder("12345", "customEvent1", "Custom Event Text 1")
	testingutils.Equals(t, "12345", builder.DeviceID())

	builder.SetDeviceID("99192")
	testingutils.Equals(t, "99192", builder.DeviceID())
}

func TestEventBuilder_Type(t *testing.T) {
	builder := c8y.NewEventBuilder("12345", "customEvent1", "Custom Event Text 1")
	testingutils.Equals(t, "customEvent1", builder.Type())

	builder.SetType("anotherType")
	testingutils.Equals(t, "anotherType", builder.Type())
}

func TestEventBuilder_GetSet(t *testing.T) {
	builder := c8y.NewEventBuilder("12345", "customEvent1", "Custom Event Text 1")

	builder.Set("c8y_CustomFragment", int64(2))
	val, ok := builder.Get("c8y_CustomFragment")
	testingutils.Equals(t, true, ok)
	testingutils.Equals(t, int64(2), val.(int64))

	_, ok = builder.Get("c8y_NonExistentProp")
	testingutils.Equals(t, false, ok)
}
