package c8y_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
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
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Assert(t, event != nil, "Event object should not be empty")

	// Get the event
	if event != nil {
		event2, resp, err := client.Event.GetEvent(context.Background(), event.ID)
		testingutils.Ok(t, err)
		testingutils.Equals(t, http.StatusOK, resp.StatusCode())
		testingutils.Equals(t, event.ID, event2.ID)
	}
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
			Source: testDevice.ID,
			Type:   eventType,
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, 3, len(col.Events))
}

func TestEventService_Update(t *testing.T) {
	client := createTestClient()

	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)

	createEvent := eventFactory(client, testDevice.ID, "testevent1")

	event1, resp, err := createEvent()
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())

	event2, resp, err := client.Event.Update(
		context.Background(),
		event1.ID,
		map[string]string{
			"text": "My new text label",
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, "My new text label", event2.Text)
}

func TestEventService_Delete(t *testing.T) {
	client := createTestClient()

	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)

	createEventType := eventFactory(client, testDevice.ID, "testevent1")
	event1, resp, err := createEventType()

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Assert(t, event1.ID != "", "event.ID should not be empty")

	resp, err = client.Event.Delete(
		context.Background(),
		event1.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())

	event2, resp, err := client.Event.GetEvent(
		context.Background(),
		event1.ID,
	)
	testingutils.Assert(t, err != nil, "Should throw an error")
	testingutils.Equals(t, http.StatusNotFound, resp.StatusCode())
	testingutils.Equals(t, "", event2.ID)

}

func TestEventService_DeleteEvents(t *testing.T) {
	client := createTestClient()

	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)

	eventType1 := "testevent1"
	eventType2 := "testevent2"

	createEventType1 := eventFactory(client, testDevice.ID, eventType1)
	createEventType2 := eventFactory(client, testDevice.ID, eventType2)

	createEventType1()
	createEventType1()
	createEventType1()
	createEventType2()

	col, _, err := client.Event.GetEvents(
		context.Background(),
		&c8y.EventCollectionOptions{
			Source: testDevice.ID,
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, 4, len(col.Events))

	resp, err := client.Event.DeleteEvents(
		context.Background(),
		&c8y.EventCollectionOptions{
			Type:   eventType1,
			Source: testDevice.ID,
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())

	col, resp, err = client.Event.GetEvents(
		context.Background(),
		&c8y.EventCollectionOptions{
			Source: testDevice.ID,
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, 1, len(col.Events))
	testingutils.Equals(t, eventType2, col.Events[0].Type)
}

func TestEventService_CreateBinary(t *testing.T) {
	client := createTestClient()

	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)

	value1 := c8y.Event{
		Time:   c8y.NewTimestamp(),
		Type:   "testevent",
		Text:   "Test Event",
		Source: c8y.NewSource(testDevice.ID),
	}

	event1, resp, err := client.Event.Create(
		context.Background(),
		value1,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Assert(t, event1.ID != "", "ID should not be empty")

	//
	// Upload file to event
	testfile1 := NewDummyFile("testfile1", "test contents 1")
	binaryobj1, resp, err := client.Event.CreateBinary(
		context.Background(),
		testfile1,
		event1.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Equals(t, event1.ID, binaryobj1.Source)
	testingutils.Assert(t, binaryobj1.Self != "", "Self link should be set")

	//
	// Download file
	downloadedFile1, err := client.Event.DownloadBinary(
		context.Background(),
		event1.ID,
	)
	testingutils.Ok(t, err)
	testingutils.FileEquals(t, testfile1, downloadedFile1)

	//
	// Remove file
	resp, err = client.Event.DeleteBinary(
		context.Background(),
		event1.ID,
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())

	//
	// Check if binary has been deleted
	downloadedFile2, err := client.Event.DownloadBinary(
		context.Background(),
		event1.ID,
	)
	testingutils.Assert(t, err != nil, "An error should be thrown if the binary does not exist")
	testingutils.Equals(t, "", downloadedFile2)
}
