package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/measurements"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
)

func Test_Measurements_DeviceResolver_ByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	// Test using device name resolver
	opts := measurements.ListOptions{
		Source: client.Measurements.DeviceResolver.ByName("device01"),
	}

	result := client.Measurements.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed measurements with device name resolver")
}

func Test_Measurements_DeviceResolver_ByExternalID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	// Test using external ID resolver
	opts := measurements.ListOptions{
		Source: client.Measurements.DeviceResolver.ByExternalID("c8y_Serial", "ABC123"),
	}

	result := client.Measurements.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed measurements with external ID resolver")
}

func Test_Measurements_DeviceResolver_DirectID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	// Test with direct ID (no resolution needed)
	opts := measurements.ListOptions{
		Source: "12345",
	}

	result := client.Measurements.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed measurements with direct ID")
}

func Test_Measurements_DeviceResolver_StringBased(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	// Test using string-based resolver (like user might pass from CLI)
	opts := measurements.ListOptions{
		Source: "name:device01",
	}

	result := client.Measurements.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed measurements with string-based name resolver")
}

func Test_Measurements_DeviceResolver_ByQuery(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	// Test using query resolver
	opts := measurements.ListOptions{
		Source: client.Measurements.DeviceResolver.ByQuery("type eq 'c8y_Device'"),
	}

	result := client.Measurements.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed measurements with query resolver")
}
