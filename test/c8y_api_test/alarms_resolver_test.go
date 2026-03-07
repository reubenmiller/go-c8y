package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
)

func Test_Alarms_DeviceResolver_ByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

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
	ctx := api.WithMockResponses(context.Background(), true)

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
	ctx := api.WithMockResponses(context.Background(), true)

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
	ctx := api.WithMockResponses(context.Background(), true)

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
	ctx := api.WithMockResponses(context.Background(), true)

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
	ctx := api.WithMockResponses(context.Background(), true)

	opts := alarms.CountOptions{
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
	ctx := api.WithMockResponses(context.Background(), true)

	opts := alarms.CountOptions{
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
	ctx := api.WithMockResponses(context.Background(), true)

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
	ctx := api.WithMockResponses(context.Background(), true)

	opts := alarms.DeleteListOptions{
		Source: "name:device01",
	}

	result := client.Alarms.DeleteList(ctx, opts)
	if result.Err != nil {
		t.Fatalf("DeleteList failed: %v", result.Err)
	}

	t.Logf("Successfully bulk deleted alarms with device resolver")
}

func Test_Alarms_Create_WithResolver_Metadata(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)
	deferredCtx := api.WithDeferredExecution(ctx, true)

	opts := alarms.CreateOptions{
		Source:   client.Alarms.DeviceResolver.ByName("device01"),
		Type:     "c8y_TestAlarm",
		Text:     "Test alarm",
		Severity: "MAJOR",
		AdditionalProperties: map[string]any{
			"custom": "value",
		},
	}

	result := client.Alarms.Create(deferredCtx, opts)
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

	t.Logf("Successfully created alarm with device resolver and captured metadata: id=%v, name=%v",
		result.Meta["id"], result.Meta["name"])

	// Execute deferred result
	result.Execute(context.Background())
	if result.Err != nil {
		t.Fatalf("Execute failed: %v", result.Err)
	}

	t.Logf("Successfully executed deferred alarm creation")
}

func Test_Alarms_Create_WithStringResolver_Metadata(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)
	deferredCtx := api.WithDeferredExecution(ctx, true)

	opts := alarms.CreateOptions{
		Source:   "name:device01",
		Type:     "c8y_TestAlarm",
		Text:     "Test alarm",
		Severity: "CRITICAL",
	}

	result := client.Alarms.Create(deferredCtx, opts)
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

	t.Logf("Successfully created alarm with string-based resolver and captured metadata")
}
