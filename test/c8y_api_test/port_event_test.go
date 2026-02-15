package api_test

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/events"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/events/eventbinaries"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func TestPortEventService_CreateEvent(t *testing.T) {
	ctx := context.Background()
	client := testcore.CreateTestClient(t)

	deviceResult := testcore.CreateDevice(t, client)
	device := deviceResult.Data
	assert.NoError(t, deviceResult.Err)

	// Create event
	event := map[string]any{
		"type": "testEvent",
		"text": "Test Event",
		"time": time.Now(),
		"source": map[string]any{
			"id": device.ID(),
		},
	}

	result := client.Events.Create(ctx, event)
	assert.NoError(t, result.Err)
	assert.NotEmpty(t, result.Data.ID())
	assert.Equal(t, "testEvent", result.Data.Type())

	// Retrieve event
	result2 := client.Events.Get(ctx, result.Data.ID())
	assert.NoError(t, result2.Err)
	assert.Equal(t, result.Data.ID(), result2.Data.ID())
	assert.Equal(t, "testEvent", result2.Data.Type())
}

func TestPortEventService_GetEvents(t *testing.T) {
	ctx := context.Background()
	client := testcore.CreateTestClient(t)

	deviceResult := testcore.CreateDevice(t, client)
	device := deviceResult.Data
	assert.NoError(t, deviceResult.Err)

	// Create test events
	for i := 0; i < 3; i++ {
		testcore.CreateEvent(t, client, &device)
	}

	// List events
	result := client.Events.List(ctx, events.ListOptions{
		Source: device.ID(),
	})
	eventList, err := op.ToSliceR(result)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(eventList), 3, "Should have at least 3 events")

	// Verify pagination
	result2 := client.Events.List(ctx, events.ListOptions{
		Source: device.ID(),
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 2,
		},
	})
	page1, err := op.ToSliceR(result2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(page1), "First page should have exactly 2 events")
}

func TestPortEventService_Update(t *testing.T) {
	ctx := context.Background()
	client := testcore.CreateTestClient(t)

	deviceResult := testcore.CreateDevice(t, client)
	device := deviceResult.Data
	assert.NoError(t, deviceResult.Err)

	// Create event
	eventResult := testcore.CreateEvent(t, client, &device)
	event1 := eventResult.Data
	assert.NoError(t, eventResult.Err)

	// Update event
	updateBody := map[string]any{
		"text": "My new text label",
	}

	result2 := client.Events.Update(ctx, event1.ID(), updateBody)
	assert.NoError(t, result2.Err)
	assert.Equal(t, "My new text label", result2.Data.Get("text").String())
}

func TestPortEventService_Delete(t *testing.T) {
	ctx := context.Background()
	client := testcore.CreateTestClient(t)

	deviceResult := testcore.CreateDevice(t, client)
	device := deviceResult.Data
	assert.NoError(t, deviceResult.Err)

	// Create event
	eventResult := testcore.CreateEvent(t, client, &device)
	event1 := eventResult.Data
	assert.NoError(t, eventResult.Err)
	assert.NotEmpty(t, event1.ID())

	// Delete event
	deleteResult := client.Events.Delete(ctx, event1.ID())
	assert.NoError(t, deleteResult.Err)

	// Verify event is deleted
	getResult := client.Events.Get(ctx, event1.ID())
	assert.Error(t, getResult.Err, "Should throw an error when getting deleted event")
	assert.Equal(t, http.StatusNotFound, getResult.HTTPStatus)
}

func TestPortEventService_DeleteEvents(t *testing.T) {
	ctx := context.Background()
	client := testcore.CreateTestClient(t)

	deviceResult := testcore.CreateDevice(t, client)
	device := deviceResult.Data
	assert.NoError(t, deviceResult.Err)

	eventType1 := "testEvent1"
	eventType2 := "testEvent2"

	// Create test events
	createEvent := func(eventType string) {
		event := map[string]any{
			"type": eventType,
			"text": "Test Event",
			"time": time.Now(),
			"source": map[string]any{
				"id": device.ID(),
			},
		}
		result := client.Events.Create(ctx, event)
		assert.NoError(t, result.Err)
	}

	createEvent(eventType1)
	createEvent(eventType1)
	createEvent(eventType1)
	createEvent(eventType2)

	// Verify 4 events exist
	result := client.Events.List(ctx, events.ListOptions{
		Source: device.ID(),
	})
	allEvents, err := op.ToSliceR(result)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(allEvents))

	// Delete events of type1
	deleteResult := client.Events.DeleteList(ctx, events.DeleteListOptions{
		Type:   eventType1,
		Source: device.ID(),
	})
	assert.NoError(t, deleteResult.Err)

	// Wait for events to be deleted
	// TODO: Add dynamic retry
	time.Sleep(1 * time.Second)

	// Verify only type2 event remains
	result2 := client.Events.List(ctx, events.ListOptions{
		Source: device.ID(),
	})
	remainingEvents, err := op.ToSliceR(result2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(remainingEvents))
	assert.Equal(t, eventType2, remainingEvents[0].Type())
}

func TestPortEventService_CreateBinary(t *testing.T) {
	ctx := context.Background()
	client := testcore.CreateTestClient(t)

	device := testcore.CreateDevice(t, client).Data

	// Create event
	value1 := model.Event{
		Time:   time.Now(),
		Type:   "testEvent",
		Text:   "Test Event",
		Source: model.NewSource(device.ID()),
	}

	event1 := client.Events.Create(ctx, value1)
	assert.NoError(t, event1.Err)
	assert.Equal(t, http.StatusCreated, event1.HTTPStatus)
	assert.NotEmpty(t, event1.Data.ID(), "ID should not be empty")

	//
	// Upload file to event
	testfile1 := testcore.NewDummyFile(t, "testFile1.txt", "test contents 1")

	binaryResult := client.Events.Binaries.Create(
		ctx,
		event1.Data.ID(),
		eventbinaries.UploadFileOptions{
			FilePath: testfile1,
		},
	)
	assert.NoError(t, binaryResult.Err)
	assert.Equal(t, http.StatusCreated, binaryResult.HTTPStatus)
	assert.NotEmpty(t, binaryResult.Data.Self(), "Self link should be set")

	//
	// Download file
	downloadResult := client.Events.Binaries.Get(ctx, event1.Data.ID())
	assert.NoError(t, downloadResult.Err)
	assert.Equal(t, http.StatusOK, downloadResult.HTTPStatus)

	// Save downloaded content to temp file for comparison
	downloadedFile := filepath.Join(t.TempDir(), "testFile1_downloaded.txt")
	testingutils.SaveBinaryToFile(t, downloadResult.Data.Reader(), downloadedFile)
	testingutils.FileEquals(t, testfile1, downloadedFile)

	//
	// Remove file
	deleteResult := client.Events.Binaries.Delete(ctx, event1.Data.ID())
	assert.NoError(t, deleteResult.Err)
	assert.Equal(t, http.StatusNoContent, deleteResult.HTTPStatus)

	//
	// Check if binary has been deleted
	getAfterDelete := client.Events.Binaries.Get(ctx, event1.Data.ID())
	assert.Error(t, getAfterDelete.Err, "An error should be thrown if the binary does not exist")
	assert.Equal(t, http.StatusNotFound, getAfterDelete.HTTPStatus)
}
