package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alarms"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
)

func Test_Alarms_DeviceResolver_ByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := c8y_api.WithMockResponses(context.Background(), true)

	opts := alarms.ListOptions{
		Source: client.Alarms.DeviceResolver.ByName("device01"),
	}

	result := client.Alarms.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed alarms with device name resolver")
}

func Test_Alarms_DeviceResolver_ByExternalID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := c8y_api.WithMockResponses(context.Background(), true)

	opts := alarms.ListOptions{
		Source: client.Alarms.DeviceResolver.ByExternalID("c8y_Serial", "ABC123"),
	}

	result := client.Alarms.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed alarms with external ID resolver")
}

func Test_Alarms_DeviceResolver_DirectID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := c8y_api.WithMockResponses(context.Background(), true)

	opts := alarms.ListOptions{
		Source: "12345",
	}

	result := client.Alarms.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed alarms with direct ID")
}

func Test_Alarms_DeviceResolver_StringBased(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := c8y_api.WithMockResponses(context.Background(), true)

	opts := alarms.ListOptions{
		Source: "name:device01",
	}

	result := client.Alarms.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed alarms with string-based device resolver")
}

func Test_Alarms_DeviceResolver_ByQuery(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := c8y_api.WithMockResponses(context.Background(), true)

	opts := alarms.ListOptions{
		Source: client.Alarms.DeviceResolver.ByQuery("type eq 'c8y_Device'"),
	}

	result := client.Alarms.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed alarms with query resolver")
}

func Test_Alarms_Count_WithDeviceResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := c8y_api.WithMockResponses(context.Background(), true)

	opts := alarms.ListOptions{
		Source: client.Alarms.DeviceResolver.ByName("device01"),
	}

	result := client.Alarms.Count(ctx, opts)
	if result.Err != nil {
		t.Fatalf("Count failed: %v", result.Err)
	}

	t.Logf("Successfully counted alarms with device name resolver")
}

func Test_Alarms_Count_WithStringBasedResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := c8y_api.WithMockResponses(context.Background(), true)

	opts := alarms.ListOptions{
		Source: "name:device01",
	}

	result := client.Alarms.Count(ctx, opts)
	if result.Err != nil {
		t.Fatalf("Count failed: %v", result.Err)
	}

	t.Logf("Successfully counted alarms with string-based device resolver")
}

func Test_Alarms_UpdateList_WithDeviceResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := c8y_api.WithMockResponses(context.Background(), true)

	opts := alarms.BulkUpdateOptions{
		Source: client.Alarms.DeviceResolver.ByName("device01"),
	}
	body := map[string]string{"status": "ACKNOWLEDGED"}

	result := client.Alarms.UpdateList(ctx, opts, body)
	if result.Err != nil {
		t.Fatalf("UpdateList failed: %v", result.Err)
	}

	t.Logf("Successfully bulk updated alarms with device resolver")
}

func Test_Alarms_DeleteList_WithDeviceResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := c8y_api.WithMockResponses(context.Background(), true)

	opts := alarms.DeleteListOptions{
		Source: "name:device01",
	}

	result := client.Alarms.DeleteList(ctx, opts)
	if result.Err != nil {
		t.Fatalf("DeleteList failed: %v", result.Err)
	}

	t.Logf("Successfully bulk deleted alarms with device resolver")
}
