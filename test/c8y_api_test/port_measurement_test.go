package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/measurements"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetMeasurementSeries(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Get a device to use as source
	device := testcore.CreateDevice(t, client).Data

	// Create a simple measurement to ensure data exists
	body := map[string]any{
		"source": map[string]any{
			"id": device.ID(),
		},
		"type": "c8y_Temperature",
		"time": time.Now(),
		"c8y_Temperature": map[string]any{
			"A": map[string]any{"value": 25.5, "unit": "°C"},
			"B": map[string]any{"value": 23.0, "unit": "°C"},
		},
	}
	createResult := client.Measurements.Create(ctx, body)
	require.NoError(t, createResult.Err)
	assert.Equal(t, 201, createResult.HTTPStatus)

	// Get measurement series
	result := client.Measurements.ListSeries(ctx, measurements.ListSeriesOptions{
		Source:   device.ID(),
		Series:   []string{"c8y_Temperature.A", "c8y_Temperature.B"},
		DateFrom: time.Now().Add(-2 * 24 * time.Hour),
		DateTo:   time.Now(),
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)
	assert.True(t, result.Data.Get("series").Exists(), "series should exist")
}

func Test_GetMeasurements(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create test device
	device := testcore.CreateDevice(t, client).Data

	// Create a test measurement
	body := map[string]any{
		"source": map[string]any{
			"id": device.ID(),
		},
		"type": "TestSeries1",
		"time": time.Now(),
		"nx_TestDevice": map[string]any{
			"Series1": map[string]any{"value": 1.0, "unit": "Counter"},
		},
	}

	createResult := client.Measurements.Create(ctx, body)
	require.NoError(t, createResult.Err)
	assert.Equal(t, 201, createResult.HTTPStatus)
	assert.NotEmpty(t, createResult.Data.ID(), "Measurement ID should not be empty")

	// Get measurements list
	result := client.Measurements.List(ctx, measurements.ListOptions{
		DateFrom: time.Now().Add(-1 * 24 * time.Hour),
		Source:   device.ID(),
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	measurementList, err := op.ToSliceR(result)
	assert.NoError(t, err)

	// Check that we have at least one measurement
	require.GreaterOrEqual(t, len(measurementList), 1, "Should have at least one measurement")

	// Check the first measurement has an ID
	assert.NotEmpty(t, measurementList[0].ID(), "First measurement should have an ID")
}

func Test_CreateMeasurement(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	device := testcore.CreateDevice(t, client).Data

	body := map[string]any{
		"source": map[string]any{
			"id": device.ID(),
		},
		"type":     "c8yTest",
		"time":     time.Now(),
		"c8y_Test": map[string]any{},
		"c8y_Temperature": map[string]any{
			"A": map[string]any{"value": 1.101, "unit": "degC"},
		},
	}

	result := client.Measurements.Create(ctx, body)

	require.NoError(t, result.Err)
	assert.Equal(t, 201, result.HTTPStatus)
	assert.NotEmpty(t, result.Data.ID(), "Measurement ID should not be empty")

	// Verify the measurement data
	assert.Equal(t, device.ID(), result.Data.SourceID())
	assert.Equal(t, "c8yTest", result.Data.Type())
}

func Test_CreateMeasurementWithDifferentTypes(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	device := testcore.CreateDevice(t, client).Data

	// Helper function to create a measurement with a specific value type
	createMeasurement := func(value any) jsonmodels.Measurement {
		body := map[string]any{
			"source": map[string]any{
				"id": device.ID(),
			},
			"type":     "c8yTest",
			"time":     time.Now(),
			"c8y_Test": map[string]any{},
			"c8y_Temperature": map[string]any{
				"A": map[string]any{"value": value, "unit": "degC"},
			},
		}

		result := client.Measurements.Create(ctx, body)
		require.NoError(t, result.Err)
		assert.Equal(t, 201, result.HTTPStatus)

		// Verify the value was stored correctly
		path := "c8y_Temperature.A.value"

		switch v := value.(type) {
		case []byte:
			actualVal := result.Data.Get(path).String()
			assert.Equal(t, string(v), actualVal)
		case []rune:
			actualVal := result.Data.Get(path).String()
			assert.Equal(t, string(v), actualVal)
		case string:
			actualVal := result.Data.Get(path).String()
			assert.Equal(t, v, actualVal)
		case bool:
			actualVal := result.Data.Get(path).Bool()
			assert.Equal(t, v, actualVal)
		case int:
			actualVal := result.Data.Get(path).Int()
			assert.Equal(t, int64(v), actualVal)
		case int8:
			actualVal := result.Data.Get(path).Int()
			assert.Equal(t, int64(v), actualVal)
		case int16:
			actualVal := result.Data.Get(path).Int()
			assert.Equal(t, int64(v), actualVal)
		case int32:
			actualVal := result.Data.Get(path).Int()
			assert.Equal(t, int64(v), actualVal)
		case int64:
			actualVal := result.Data.Get(path).Int()
			assert.Equal(t, v, actualVal)
		case uint:
			actualVal := result.Data.Get(path).Uint()
			assert.Equal(t, uint64(v), actualVal)
		case uint8:
			actualVal := result.Data.Get(path).Uint()
			assert.Equal(t, uint64(v), actualVal)
		case uint16:
			actualVal := result.Data.Get(path).Uint()
			assert.Equal(t, uint64(v), actualVal)
		case uint32:
			actualVal := result.Data.Get(path).Uint()
			assert.Equal(t, uint64(v), actualVal)
		case uint64:
			actualVal := result.Data.Get(path).Uint()
			assert.Equal(t, v, actualVal)
		case float32:
			actualVal := float32(result.Data.Get(path).Float())
			assert.EqualValues(t, v, actualVal)
		case float64:
			actualVal := result.Data.Get(path).Float()
			assert.EqualValues(t, v, actualVal)
		default:
			t.Errorf("Unsupported data type")
		}

		return result.Data
	}

	// Test various numeric types
	createMeasurement(float64(1.64))
	createMeasurement(float32(1.32))

	// integer values
	createMeasurement(int64(64))
	createMeasurement(int32(32))
	createMeasurement(int16(16))
	createMeasurement(int8(8))
	createMeasurement(int(101))

	// unsigned integer values
	createMeasurement(uint64(64))
	createMeasurement(uint32(32))
	createMeasurement(uint16(16))
	createMeasurement(uint8(8))
	createMeasurement(uint(101))
}

func Test_DeleteMeasurements(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	device := testcore.CreateDevice(t, client).Data
	valueFragmentType := "nx_Type1_" + testingutils.RandomString(5)

	// Create test measurements
	body1 := map[string]any{
		"source": map[string]any{
			"id": device.ID(),
		},
		"type": "TestSeries1",
		"time": time.Now(),
		valueFragmentType: map[string]any{
			"Variable1": map[string]any{"value": 1.0, "unit": "Counter"},
		},
	}
	createResult1 := client.Measurements.Create(ctx, body1)
	require.NoError(t, createResult1.Err)
	assert.Equal(t, 201, createResult1.HTTPStatus)
	assert.NotEmpty(t, createResult1.Data.ID(), "Measurement ID should not be empty")

	body2 := map[string]any{
		"source": map[string]any{
			"id": device.ID(),
		},
		"type": "TestSeries1",
		"time": time.Now(),
		valueFragmentType: map[string]any{
			"Variable2": map[string]any{"value": 2.0, "unit": "Counter"},
		},
	}
	createResult2 := client.Measurements.Create(ctx, body2)
	require.NoError(t, createResult2.Err)
	assert.Equal(t, 201, createResult2.HTTPStatus)
	assert.NotEmpty(t, createResult2.Data.ID(), "Measurement ID should not be empty")

	// Wait for measurements to be created
	time.Sleep(2 * time.Second)

	// Verify measurements exist
	listResult := client.Measurements.List(ctx, measurements.ListOptions{
		Source:            device.ID(),
		ValueFragmentType: valueFragmentType,
	})
	require.NoError(t, listResult.Err)
	assert.Equal(t, 200, listResult.HTTPStatus)
	assert.EqualValues(t, 2, listResult.Data.Length())

	// Delete the measurements
	deleteResult := client.Measurements.DeleteList(ctx, measurements.DeleteListOptions{
		Source:       device.ID(),
		FragmentType: valueFragmentType,
	})
	require.NoError(t, deleteResult.Err)
	assert.Equal(t, 204, deleteResult.HTTPStatus)

	// Wait for deletion to complete
	time.Sleep(2 * time.Second)

	// Verify measurements have been deleted
	listResultAfter := client.Measurements.List(ctx, measurements.ListOptions{
		Source:            device.ID(),
		ValueFragmentType: valueFragmentType,
	})
	require.NoError(t, listResultAfter.Err)
	assert.Equal(t, 200, listResultAfter.HTTPStatus)
	assert.EqualValues(t, 0, listResultAfter.Data.Length())
}

func Test_CreateMeasurements(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	client.SetDebug(true)

	device := testcore.CreateDevice(t, client).Data
	valueFragmentType := "nx_common"

	// Create multiple measurements in a single request
	body := map[string]any{
		"measurements": []map[string]any{
			{
				"source": map[string]any{
					"id": device.ID(),
				},
				"type": "TestSeries1",
				"time": time.Now(),
				valueFragmentType: map[string]any{
					"Signal1": map[string]any{"value": 1.1, "unit": "Counter"},
				},
			},
			{
				"source": map[string]any{
					"id": device.ID(),
				},
				"type": "TestSeries2",
				"time": time.Now(),
				valueFragmentType: map[string]any{
					"Signal2": map[string]any{"value": 2.0, "unit": "Counter"},
				},
			},
		},
	}

	result := client.Measurements.CreateList(ctx, body)
	require.NoError(t, result.Err)
	assert.Equal(t, 201, result.HTTPStatus)

	// Verify that 2 measurements were created
	assert.EqualValues(t, 2, result.Data.Length())
}
