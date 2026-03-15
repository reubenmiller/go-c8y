package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/events"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
)

func Test_Events_Create_WithResolver_Metadata(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)
	deferredCtx := api.WithDeferredExecution(ctx, true)

	opts := events.CreateOptions{
		Source: client.Events.DeviceResolver.ByName("device01"),
		Type:   "c8y_TestEvent",
		Text:   "Test event",
		Time:   time.Now(),
		AdditionalProperties: map[string]any{
			"custom": "value",
		},
	}

	result := client.Events.Create(deferredCtx, opts)
	if result.Err != nil {
		t.Fatalf("Create failed: %v", result.Err)
	}

	// Check that metadata is populated
	if result.Meta["id"] == nil {
		t.Errorf("Expected 'id' in metadata, but it was nil")
	}
	if result.Meta["name"] == nil {
		t.Errorf("Expected 'name' in metadata, but it was nil")
	}

	t.Logf("Successfully created event with device resolver and captured metadata: id=%v, name=%v",
		result.Meta["id"], result.Meta["name"])

	// Execute deferred result
	result.Execute(context.Background())
	if result.Err != nil {
		t.Fatalf("Execute failed: %v", result.Err)
	}

	t.Logf("Successfully executed deferred event creation")
}

func Test_Events_Create_WithStringResolver_Metadata(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)
	deferredCtx := api.WithDeferredExecution(ctx, true)

	opts := events.CreateOptions{
		Source: managedobjects.ByName("device01"),
		Type:   "c8y_TestEvent",
		Text:   "Test event",
		Time:   time.Now(),
	}

	result := client.Events.Create(deferredCtx, opts)
	if result.Err != nil {
		t.Fatalf("Create failed: %v", result.Err)
	}

	// Check that metadata is populated
	if result.Meta["id"] == nil {
		t.Errorf("Expected 'id' in metadata, but it was nil")
	}
	if result.Meta["name"] == nil {
		t.Errorf("Expected 'name' in metadata, but it was nil")
	}

	t.Logf("Successfully created event with string-based resolver and captured metadata")
}

func Test_Events_Create_WithAdditionalProperties(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	type CustomEvent struct {
		Position map[string]any `json:"c8y_Position"`
	}

	opts := events.CreateOptions{
		Source: managedobjects.ByID("12345"),
		Type:   "c8y_LocationUpdate",
		Text:   "Location updated",
		Time:   time.Now(),
		AdditionalProperties: CustomEvent{
			Position: map[string]any{
				"lat": 51.5074,
				"lng": -0.1278,
				"alt": 100.0,
			},
		},
	}

	result := client.Events.Create(ctx, opts)
	if result.Err != nil {
		t.Fatalf("Create failed: %v", result.Err)
	}

	// With mock responses, we can't verify the actual merged body,
	// but we can verify the operation completed without error
	t.Logf("Successfully created event with merged custom properties")
}
