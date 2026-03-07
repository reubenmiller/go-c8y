package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/measurements"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_Measurements_ListSeries_WithResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	opts := measurements.ListSeriesOptions{
		Source: client.Measurements.DeviceResolver.ByName("device01"),
		Series: []string{"c8y_Temperature.T", "c8y_Humidity.H"},
	}

	result := client.Measurements.ListSeries(ctx, opts)
	if result.Err != nil {
		t.Fatalf("ListSeries failed: %v", result.Err)
	}

	count := 0
	series := result.Data.GetSeriesNames()
	assert.GreaterOrEqual(t, len(series), 2)
	for _, item := range result.Data.ToTabular() {
		assert.NotZero(t, item.Time)
		assert.Len(t, item.Values, 2)
		assert.Greater(t, item.Values[0].GetMax(), float64(0))
		assert.Greater(t, item.Values[0].GetMin(), float64(0))
		count += 1
	}

	assert.GreaterOrEqual(t, count, 1)

	t.Logf("Successfully retrieved measurement series")
}

func Test_Measurements_ListSeries_ToTabular(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	opts := measurements.ListSeriesOptions{
		Source: "12345",
		Series: []string{"c8y_Temperature.T"},
	}

	result := client.Measurements.ListSeries(ctx, opts)
	if result.Err != nil {
		t.Fatalf("ListSeries failed: %v", result.Err)
	}

	// Test the tabular transformation
	series := result.Data
	rows := series.ToTabular()
	t.Logf("Transformed to %d rows", len(rows))

	// Test getting series names
	names := series.GetSeriesNames()
	t.Logf("Series names: %v", names)

	// Verify device info is populated
	t.Logf("DeviceID: %s, DeviceName: %s", series.DeviceID, series.DeviceName)
	if series.DeviceID == "" {
		t.Error("Expected DeviceID to be populated")
	}
}
