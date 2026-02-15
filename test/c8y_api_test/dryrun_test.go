package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/events"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/operations"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_DryRun_ManagedObject_Get(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Enable dry run
	ctx := api.WithDryRun(context.Background(), true)

	// Get a managed object (will return mock response)
	result := client.ManagedObjects.Get(ctx, "12345", managedobjects.GetOptions{})

	// Should not error
	assert.NoError(t, result.Err)

	// Should have mock data
	assert.NotEmpty(t, result.Data.ID())
	assert.Equal(t, "12345", result.Data.ID())
	assert.Equal(t, "Mock Device", result.Data.Name())
	assert.Equal(t, "c8y_Device", result.Data.Type())

	// Should have dry run header
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
}

func Test_DryRun_ManagedObject_List(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Enable dry run
	ctx := api.WithDryRun(context.Background(), true)

	// List managed objects (will return mock collection)
	result := client.ManagedObjects.List(ctx, managedobjects.ListOptions{})

	// Should not error
	assert.NoError(t, result.Err)

	// Should have mock collection data
	assert.Greater(t, result.Data.Length(), 0)

	// Verify we can iterate
	count := 0
	for range result.Data.Iter() {
		count++
	}
	assert.Equal(t, 2, count)
}

func Test_DryRun_ManagedObject_Create(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Enable dry run
	ctx := api.WithDryRun(context.Background(), true)

	// Create a managed object (will return mock response)
	result := client.ManagedObjects.Create(ctx, map[string]any{
		"name": testingutils.RandomString(16),
		"type": "c8y_TestDevice",
	})

	// Should not error
	assert.NoError(t, result.Err)

	// Should return 201 Created
	assert.Equal(t, http.StatusCreated, result.HTTPStatus)

	// Should have mock ID
	assert.NotEmpty(t, result.Data.ID())
	assert.Equal(t, "12345", result.Data.ID())
}

func Test_DryRun_ManagedObject_Delete(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Enable dry run
	ctx := api.WithDryRun(context.Background(), true)

	// Delete a managed object (will return mock response)
	result := client.ManagedObjects.Delete(ctx, "12345", managedobjects.DeleteOptions{})

	// Should not error
	assert.NoError(t, result.Err)

	// Should return 204 No Content
	assert.Equal(t, http.StatusNoContent, result.HTTPStatus)
}

func Test_DryRun_Alarm_Get(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Enable dry run
	ctx := api.WithDryRun(context.Background(), true)

	// Get an alarm (will return mock response)
	result := client.Alarms.Get(ctx, "54321")

	// Should not error
	assert.NoError(t, result.Err)

	// Should have mock data
	assert.NotEmpty(t, result.Data.ID())
	assert.Equal(t, "54321", result.Data.ID())
	assert.Contains(t, result.Data.Type(), "c8y_")
}

func Test_DryRun_Alarm_List(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Enable dry run
	ctx := api.WithDryRun(context.Background(), true)

	// List alarms (will return mock collection)
	result := client.Alarms.List(ctx, alarms.ListOptions{})

	// Should not error
	assert.NoError(t, result.Err)

	// Should have mock collection data
	assert.Greater(t, result.Data.Length(), 0)
}

func Test_DryRun_Event_Get(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Enable dry run
	ctx := api.WithDryRun(context.Background(), true)

	// Get an event (will return mock response)
	result := client.Events.Get(ctx, "98765")

	// Should not error
	assert.NoError(t, result.Err)

	// Should have mock data
	assert.NotEmpty(t, result.Data.ID())
	assert.Equal(t, "98765", result.Data.ID())
}

func Test_DryRun_Event_List(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Enable dry run
	ctx := api.WithDryRun(context.Background(), true)

	// List events (will return mock collection)
	result := client.Events.List(ctx, events.ListOptions{})

	// Should not error
	assert.NoError(t, result.Err)

	// Should have mock collection data
	assert.Greater(t, result.Data.Length(), 0)
}

func Test_DryRun_Operation_Get(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Enable dry run
	ctx := api.WithDryRun(context.Background(), true)

	// Get an operation (will return mock response)
	result := client.Operations.Get(ctx, "11111")

	// Should not error
	assert.NoError(t, result.Err)

	// Should have mock data
	assert.NotEmpty(t, result.Data.ID())
	assert.Equal(t, "11111", result.Data.ID())
}

func Test_DryRun_Operation_List(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Enable dry run
	ctx := api.WithDryRun(context.Background(), true)

	// List operations (will return mock collection)
	result := client.Operations.List(ctx, operations.ListOptions{})

	// Should not error
	assert.NoError(t, result.Err)

	// Should have mock collection data
	assert.Greater(t, result.Data.Length(), 0)
}

func Test_DryRun_Disabled_RealRequest(t *testing.T) {
	t.Skip("Skipping test that would make real API request")
	client := testcore.CreateTestClient(t)

	// Normal context (dry run disabled)
	ctx := context.Background()

	// Verify dry run is disabled
	assert.False(t, api.IsDryRun(ctx))

	// This would make a real request (but will likely fail without proper setup)
	// We just verify that dry run is not enabled
	result := client.ManagedObjects.Get(ctx, "12345", managedobjects.GetOptions{})

	// Will error because it tries to make a real request
	// This is expected behavior - just verifying dry run is not interfering
	_ = result
}

func Test_DryRun_ContextCheck(t *testing.T) {
	// Test the helper functions
	normalCtx := context.Background()
	dryRunCtx := api.WithDryRun(context.Background(), true)

	assert.False(t, api.IsDryRun(normalCtx))
	assert.True(t, api.IsDryRun(dryRunCtx))

	// Test disabling dry run
	disabledCtx := api.WithDryRun(context.Background(), false)
	assert.False(t, api.IsDryRun(disabledCtx))
}
