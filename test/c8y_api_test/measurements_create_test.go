package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/measurements"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
)

func Test_Measurements_Create_WithResolver_Metadata(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)
	deferredCtx := api.WithDeferredExecution(ctx, true)

	opts := measurements.CreateOptions{
		Source: client.Measurements.DeviceResolver.ByName("device01"),
		Type:   "c8y_Temperature",
		Time:   time.Now(),
		AdditionalProperties: map[string]any{
			"c8y_Temperature": map[string]any{
				"T": map[string]any{
					"value": 23.5,
					"unit":  "°C",
				},
			},
		},
	}

	result := client.Measurements.Create(deferredCtx, opts)
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

	t.Logf("Successfully created measurement with device resolver and captured metadata: id=%v, name=%v",
		result.Meta["id"], result.Meta["name"])

	// Execute deferred result
	result.Execute(context.Background())
	if result.Err != nil {
		t.Fatalf("Execute failed: %v", result.Err)
	}

	t.Logf("Successfully executed deferred measurement creation")
}

func Test_Measurements_Create_WithStringResolver_Metadata(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)
	deferredCtx := api.WithDeferredExecution(ctx, true)

	opts := measurements.CreateOptions{
		Source: "query:type eq 'c8y_Device'",
		Type:   "c8y_Humidity",
		Time:   time.Now(),
		AdditionalProperties: map[string]any{
			"c8y_Humidity": map[string]any{
				"H": map[string]any{
					"value": 65.0,
					"unit":  "%",
				},
			},
		},
	}

	result := client.Measurements.Create(deferredCtx, opts)
	if result.Err != nil {
		t.Fatalf("Create failed: %v", result.Err)
	}

	// Check that metadata is populated
	if result.Meta["id"] == nil {
		t.Errorf("Expected 'id' in metadata, but it was nil")
	}

	t.Logf("Successfully created measurement with string-based resolver and captured metadata")
}

func Test_Measurements_Create_WithMultipleFragments(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	type MultiMeasurement struct {
		Temperature map[string]any `json:"c8y_Temperature"`
		Humidity    map[string]any `json:"c8y_Humidity"`
		Pressure    map[string]any `json:"c8y_Pressure"`
	}

	opts := measurements.CreateOptions{
		Source: "12345",
		Type:   "c8y_EnvironmentalMeasurement",
		Time:   time.Now(),
		AdditionalProperties: MultiMeasurement{
			Temperature: map[string]any{
				"T": map[string]any{"value": 23.5, "unit": "°C"},
			},
			Humidity: map[string]any{
				"H": map[string]any{"value": 65.0, "unit": "%"},
			},
			Pressure: map[string]any{
				"P": map[string]any{"value": 1013.25, "unit": "hPa"},
			},
		},
	}

	result := client.Measurements.Create(ctx, opts)
	if result.Err != nil {
		t.Fatalf("Create failed: %v", result.Err)
	}

	// With mock responses, we can't verify the actual merged body,
	// but we can verify the operation completed without error
	t.Logf("Successfully created measurement with multiple merged fragments")
}
