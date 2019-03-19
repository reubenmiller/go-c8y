package c8y_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/c8y_test/testingutils"

	c8y "github.com/reubenmiller/go-c8y"
)

func eventFactory(client *c8y.Client, deviceID string, eventType string) func() (*c8y.Event, *c8y.Response, error) {
	counter := 0
	return func() (*c8y.Event, *c8y.Response, error) {
		counter++
		event := c8y.Event{
			Time:   c8y.NewTimestamp(),
			Type:   eventType,
			Text:   fmt.Sprintf("Test Event %d", counter),
			Source: c8y.NewSource(deviceID),
		}
		return client.Event.Create(context.Background(), event)
	}
}

func TestEventService_CreateEvent(t *testing.T) {
	client := createTestClient()

	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)

	value := c8y.Event{
		Time:   c8y.NewTimestamp(),
		Type:   "testevent",
		Text:   "Test Event",
		Source: c8y.NewSource(testDevice.ID),
	}

	event, resp, err := client.Event.Create(context.Background(), value)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode)
	testingutils.Assert(t, event != nil, "Event object should not be empty")

	// Get the event
	event2, resp, err := client.Event.GetEvent(context.Background(), event.ID)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, event.ID, event2.ID)
}

func TestEventService_GetEvents(t *testing.T) {
	client := createTestClient()

	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)

	eventType := "testevent1"

	createEvent := eventFactory(client, testDevice.ID, eventType)

	createEvent()
	createEvent()
	createEvent()

	col, resp, err := client.Event.GetEvents(
		context.Background(),
		&c8y.EventCollectionOptions{
			Type: eventType,
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, 3, len(col.Events))
}

func TestEventService_Update(t *testing.T) {
	client := createTestClient()

	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)

	createEvent := eventFactory(client, testDevice.ID, "testevent1")

	event1, resp, err := createEvent()
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode)

	event2, resp, err := client.Event.Update(
		context.Background(),
		event1.ID,
		map[string]string{
			"text": "My new text label",
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, "My new text label", event2.Text)
}
