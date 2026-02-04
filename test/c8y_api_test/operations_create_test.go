package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/operations"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
)

func Test_Operations_Create_WithResolver_Metadata(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := c8y_api.WithMockResponses(context.Background(), true)
	deferredCtx := c8y_api.WithDeferredExecution(ctx, true)

	opts := operations.CreateOptions{
		DeviceID:    client.Operations.DeviceResolver.ByName("device01"),
		Description: "Restart device",
		AdditionalProperties: map[string]any{
			"c8y_Restart": map[string]any{},
		},
	}

	result := client.Operations.Create(deferredCtx, opts)
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

	t.Logf("Successfully created operation with device resolver and captured metadata: id=%v, name=%v",
		result.Meta["id"], result.Meta["name"])

	// Execute deferred result
	result.Execute(context.Background())
	if result.Err != nil {
		t.Fatalf("Execute failed: %v", result.Err)
	}

	t.Logf("Successfully executed deferred operation creation")
}

func Test_Operations_Create_WithStringResolver_Metadata(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := c8y_api.WithMockResponses(context.Background(), true)
	deferredCtx := c8y_api.WithDeferredExecution(ctx, true)

	opts := operations.CreateOptions{
		DeviceID:    "ext:c8y_Serial:ABC123",
		Description: "Configure device",
		AdditionalProperties: map[string]any{
			"c8y_Configuration": map[string]any{
				"config": "value",
			},
		},
	}

	result := client.Operations.Create(deferredCtx, opts)
	if result.Err != nil {
		t.Fatalf("Create failed: %v", result.Err)
	}

	// Check that metadata is populated
	if result.Meta["id"] == nil {
		t.Errorf("Expected 'id' in metadata, but it was nil")
	}

	t.Logf("Successfully created operation with string-based resolver and captured metadata")
}

func Test_Operations_Create_WithComplexPayload(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := c8y_api.WithMockResponses(context.Background(), true)

	type SoftwareUpdate struct {
		Software []map[string]any `json:"c8y_SoftwareUpdate"`
	}

	opts := operations.CreateOptions{
		DeviceID:    "12345",
		Description: "Install software",
		AdditionalProperties: SoftwareUpdate{
			Software: []map[string]any{
				{
					"name":    "package1",
					"version": "1.0.0",
					"url":     "http://example.com/package1",
				},
				{
					"name":    "package2",
					"version": "2.0.0",
					"url":     "http://example.com/package2",
				},
			},
		},
	}

	result := client.Operations.Create(ctx, opts)
	if result.Err != nil {
		t.Fatalf("Create failed: %v", result.Err)
	}

	// With mock responses, we can't verify the actual merged body,
	// but we can verify the operation completed without error
	t.Logf("Successfully created operation with complex merged properties")
}
