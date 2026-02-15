package api_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

// Test demonstrates using op.Pipe for complex sequential operations
func Test_ManagedObjectPipeline(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	createOrGetDevice2 := func(ctx context.Context, mo jsonmodels.ManagedObject) (jsonmodels.ManagedObject, error) {
		result := client.ManagedObjects.GetOrCreateByName(ctx, mo.Name(), mo.Type(), map[string]any{
			"name":         mo.Name(),
			"type":         mo.Type(),
			"c8y_IsDevice": map[string]any{},
		})
		return result.Data, result.Err
	}

	// Remaining steps are Step[ManagedObject] (same input/output type)
	addCustomProperty := func(ctx context.Context, mo jsonmodels.ManagedObject) (jsonmodels.ManagedObject, error) {
		updateResult := client.ManagedObjects.Update(ctx, mo.ID(), map[string]any{
			"customProperty": "customValue",
			"updatedAt":      time.Now().Format(time.RFC3339),
		})
		return updateResult.Data, updateResult.Err
	}

	validateDevice := func(ctx context.Context, mo jsonmodels.ManagedObject) (jsonmodels.ManagedObject, error) {
		if mo.ID() == "" {
			return mo, fmt.Errorf("device has no ID")
		}
		if mo.Name() == "" {
			return mo, fmt.Errorf("device has no name")
		}
		return mo, nil
	}

	// Build the pipeline
	pipeline := op.Pipe(
		createOrGetDevice2,
		addCustomProperty,
		op.Tap(func(ctx context.Context, mo jsonmodels.ManagedObject) error {
			t.Logf("Updated device: id=%s", mo.ID())
			return nil
		}),
		validateDevice,
	)

	// Run the complete pipeline
	deviceName := testingutils.RandomString(16)
	deviceType := testingutils.RandomString(16)

	device, err := pipeline(ctx, jsonmodels.NewManagedObjectWithOptions(deviceName, deviceType))
	assert.NoError(t, err)
	assert.NotEmpty(t, device.ID())
	assert.Equal(t, deviceName, device.Name())

	// Cleanup
	deleteResult := client.ManagedObjects.Delete(ctx, device.ID(), managedobjects.DeleteOptions{})
	assert.NoError(t, deleteResult.Err)
}

// Test demonstrates using Retry with op.Pipe and If for conditional operations
func Test_ManagedObjectPipelineWithRetry(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	deviceName := testingutils.RandomString(16)
	deviceType := testingutils.RandomString(16)

	// Create initial device
	createResult := client.ManagedObjects.Create(ctx, map[string]any{
		"name":         deviceName,
		"type":         deviceType,
		"c8y_IsDevice": map[string]any{},
	})
	assert.NoError(t, createResult.Err)
	device := createResult.Data

	updateAttempts := 0

	// Simulate an update operation that uses Retry for resilience
	updateWithRetry := op.Retry(
		func(ctx context.Context, mo jsonmodels.ManagedObject) (jsonmodels.ManagedObject, error) {
			updateAttempts++
			t.Logf("Update attempt %d for device %s", updateAttempts, mo.ID())

			result := client.ManagedObjects.Update(ctx, mo.ID(), map[string]any{
				"customProp": "value123",
				"retryCount": updateAttempts,
			})
			return result.Data, result.Err
		},
		op.RetryConfig{
			MaxAttempts:     3,
			InitialInterval: 50 * time.Millisecond,
			Multiplier:      2.0,
		},
	)

	conditionalUpdate := op.If(
		func(mo jsonmodels.ManagedObject) bool {
			// Only update if device doesn't have customProp
			return mo.Get("customProp").String() == ""
		},
		updateWithRetry,
		func(ctx context.Context, mo jsonmodels.ManagedObject) (jsonmodels.ManagedObject, error) {
			t.Logf("Device %s already has customProp, skipping update", mo.ID())
			return mo, nil
		},
	)

	// Execute pipeline with conditional retry
	pipeline := op.Pipe(
		conditionalUpdate,
		op.Tap(func(ctx context.Context, mo jsonmodels.ManagedObject) error {
			t.Logf("Final device state: customProp=%s", mo.Get("customProp").String())
			return nil
		}),
	)

	device, err := pipeline(ctx, device)
	assert.NoError(t, err)
	assert.NotEmpty(t, device.ID())
	assert.Equal(t, "value123", device.Get("customProp").String())

	// Cleanup
	deleteResult := client.ManagedObjects.Delete(ctx, device.ID(), managedobjects.DeleteOptions{})
	assert.NoError(t, deleteResult.Err)
}

// Test demonstrates using MapResult for transforming Results
func Test_ManagedObjectPipelineWithMapResult(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	deviceName := testingutils.RandomString(16)
	deviceType := testingutils.RandomString(16)

	// Create a device
	createResult := client.ManagedObjects.Create(ctx, map[string]any{
		"name":         deviceName,
		"type":         deviceType,
		"c8y_IsDevice": map[string]any{},
	})
	assert.NoError(t, createResult.Err)
	deviceID := createResult.Data.ID()

	// Use MapResult to transform Result[ManagedObject] -> Result[string]
	// Extract just the ID from the device
	getDeviceID := func(ctx context.Context, id string) (op.Result[string], error) {
		result := client.ManagedObjects.Get(ctx, id, managedobjects.GetOptions{})

		// Transform Result[ManagedObject] to Result[string]
		idResult := op.MapResult(result, func(mo jsonmodels.ManagedObject) string {
			return mo.ID()
		})

		return idResult, result.Err
	}

	idResult, err := getDeviceID(ctx, deviceID)
	assert.NoError(t, err)
	assert.Equal(t, deviceID, idResult.Data)
	assert.Equal(t, "OK", string(idResult.Status))

	// Cleanup
	deleteResult := client.ManagedObjects.Delete(ctx, deviceID, managedobjects.DeleteOptions{})
	assert.NoError(t, deleteResult.Err)
}

// Test demonstrates complex multi-step pipeline with error handling
func Test_ManagedObjectComplexPipeline(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	deviceName := testingutils.RandomString(16)
	deviceType := testingutils.RandomString(16)

	// Step 1: Get or create device
	step1 := func(ctx context.Context, name string) (jsonmodels.ManagedObject, error) {
		result := client.ManagedObjects.GetOrCreateByName(ctx, name, deviceType, map[string]any{
			"name":         name,
			"type":         deviceType,
			"c8y_IsDevice": map[string]any{},
		})
		return result.Data, result.Err
	}

	// Step 2: Add hardware info
	step2 := func(ctx context.Context, mo jsonmodels.ManagedObject) (jsonmodels.ManagedObject, error) {
		result := client.ManagedObjects.Update(ctx, mo.ID(), map[string]any{
			"c8y_Hardware": map[string]any{
				"model":        "TestModel",
				"serialNumber": testingutils.RandomString(8),
			},
		})
		return result.Data, result.Err
	}

	// Step 3: Add firmware info
	step3 := func(ctx context.Context, mo jsonmodels.ManagedObject) (jsonmodels.ManagedObject, error) {
		result := client.ManagedObjects.Update(ctx, mo.ID(), map[string]any{
			"c8y_Firmware": map[string]any{
				"name":    "firmware-v1",
				"version": "1.0.0",
				"url":     "https://example.com/firmware",
			},
		})
		return result.Data, result.Err
	}

	// Step 4: Validate all properties are set
	step4 := func(ctx context.Context, mo jsonmodels.ManagedObject) (jsonmodels.ManagedObject, error) {
		if !mo.Get("c8y_Hardware").Exists() {
			return mo, fmt.Errorf("missing c8y_Hardware")
		}
		if !mo.Get("c8y_Firmware").Exists() {
			return mo, fmt.Errorf("missing c8y_Firmware")
		}
		return mo, nil
	}

	// Step 1: Get or create device
	device, err := step1(ctx, deviceName)
	if err != nil {
		t.Fatalf("Failed to create device: %v", err)
	}
	t.Logf("✓ Step 1: Device created/found: %s", device.ID())

	// Execute complete pipeline with logging
	pipeline := op.Pipe(
		step2,
		op.Tap(func(ctx context.Context, mo jsonmodels.ManagedObject) error {
			t.Logf("✓ Step 2: Hardware info added")
			return nil
		}),
		step3,
		op.Tap(func(ctx context.Context, mo jsonmodels.ManagedObject) error {
			t.Logf("✓ Step 3: Firmware info added")
			return nil
		}),
		step4,
		op.Tap(func(ctx context.Context, mo jsonmodels.ManagedObject) error {
			t.Logf("✓ Step 4: Validation passed")
			return nil
		}),
	)

	device, err = pipeline(ctx, device)
	assert.NoError(t, err)
	assert.NotEmpty(t, device.ID())
	assert.True(t, device.Get("c8y_Hardware").Exists())
	assert.True(t, device.Get("c8y_Firmware").Exists())

	// Cleanup
	deleteResult := client.ManagedObjects.Delete(ctx, device.ID(), managedobjects.DeleteOptions{})
	assert.NoError(t, deleteResult.Err)
}
