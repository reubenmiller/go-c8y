package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/events"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
)

func Test_Events_DeviceResolver_ByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	opts := events.ListOptions{
		Source: client.Events.DeviceResolver.ByName("device01"),
	}

	result := client.Events.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed events with device name resolver")
}

func Test_Events_DeviceResolver_ByExternalID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	opts := events.ListOptions{
		Source: client.Events.DeviceResolver.ByExternalID("c8y_Serial", "ABC123"),
	}

	result := client.Events.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed events with external ID resolver")
}

func Test_Events_DeviceResolver_StringBased(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	opts := events.ListOptions{
		Source: managedobjects.ByName("device01"),
	}

	result := client.Events.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed events with string-based device resolver")
}

func Test_Events_DeviceResolver_ByQuery(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	opts := events.ListOptions{
		Source: client.Events.DeviceResolver.ByQuery("type eq 'c8y_Device'"),
	}

	result := client.Events.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed events with query resolver")
}
