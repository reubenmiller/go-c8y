package c8y_api_test

import (
	"context"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DeferredExecution_Get(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Prepare the operation without executing
	ctx := c8y_api.WithDeferredExecution(context.Background(), true)
	prepared := client.ManagedObjects.Get(ctx, "12345", managedobjects.GetOptions{})

	// Should not have executed yet
	assert.True(t, prepared.IsDeferred(), "Result should be deferred")
	assert.NotNil(t, prepared.Request, "Request should be captured for inspection")
	assert.Equal(t, "GET", prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/inventory/managedObjects/12345")

	// Now execute it (use dry run for testing to avoid real HTTP calls)
	execCtx := c8y_api.WithDryRun(context.Background(), true)
	result := prepared.Execute(execCtx)

	// Should have executed
	assert.False(t, result.IsDeferred(), "Result should not be deferred after execution")
	assert.NoError(t, result.Err)
	assert.True(t, result.IsSuccess())
}

func Test_DeferredExecution_Delete(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Prepare a DELETE operation
	ctx := c8y_api.WithDeferredExecution(context.Background(), true)
	prepared := client.ManagedObjects.Delete(ctx, "device-to-delete", managedobjects.DeleteOptions{})

	// Should be deferred
	assert.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)

	// Inspect the prepared request
	assert.Equal(t, "DELETE", prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/inventory/managedObjects/device-to-delete")

	// User can decide whether to execute or cancel
	// For this test, we'll execute it (use dry run for testing)
	execCtx := c8y_api.WithDryRun(context.Background(), true)
	result := prepared.Execute(execCtx)

	assert.False(t, result.IsDeferred())
	assert.NoError(t, result.Err)
}

func Test_DeferredExecution_Create(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Prepare a POST operation
	ctx := c8y_api.WithDeferredExecution(context.Background(), true)
	data := map[string]any{
		"name": "Test Device",
		"type": "c8y_TestDevice",
	}
	prepared := client.ManagedObjects.Create(ctx, data)

	// Should be deferred
	assert.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)

	// Inspect the request
	assert.Equal(t, "POST", prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/inventory/managedObjects")

	// Execute (use dry run for testing)
	execCtx := c8y_api.WithDryRun(context.Background(), true)
	result := prepared.Execute(execCtx)
	assert.False(t, result.IsDeferred())
	assert.NoError(t, result.Err)
}

func Test_DeferredExecution_List(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Prepare a LIST operation
	ctx := c8y_api.WithDeferredExecution(context.Background(), true)
	prepared := client.ManagedObjects.List(ctx, managedobjects.ListOptions{
		Query: "name eq 'test*'",
	})

	// Should be deferred
	assert.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)

	// Inspect
	assert.Equal(t, "GET", prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/inventory/managedObjects")
	assert.Contains(t, prepared.Request.URL.RawQuery, "query=")

	// Execute (use dry run for testing)
	execCtx := c8y_api.WithDryRun(context.Background(), true)
	result := prepared.Execute(execCtx)
	assert.False(t, result.IsDeferred())
	assert.NoError(t, result.Err)
}

func Test_DeferredExecution_Cancel(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Prepare a DELETE operation
	ctx := c8y_api.WithDeferredExecution(context.Background(), true)
	prepared := client.ManagedObjects.Delete(ctx, "device-id", managedobjects.DeleteOptions{})

	// Inspect and decide to cancel
	assert.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)

	// User decides not to execute - just don't call Execute()
	// The operation is cancelled by not proceeding
	assert.Contains(t, prepared.Request.URL.Path, "device-id")

	// No execution happens - test passes without calling Execute()
}

func Test_DeferredExecution_AlreadyExecuted(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Normal execution (not deferred)
	ctx := context.Background()
	result := client.ManagedObjects.Get(ctx, "12345", managedobjects.GetOptions{})

	// Should not be deferred
	assert.False(t, result.IsDeferred())

	// Calling Execute on an already-executed result should return itself
	result2 := result.Execute(ctx)
	assert.Equal(t, result.Status, result2.Status)
	assert.False(t, result2.IsDeferred())
}

func Test_DeferredExecution_ParameterResolution(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// This tests that deferred execution still resolves parameters
	// (e.g., device name -> ID lookup via source.Resolver)
	ctx := c8y_api.WithDeferredExecution(context.Background(), true)

	// In a real scenario, this might resolve "device-name" to an ID
	prepared := client.ManagedObjects.Delete(ctx, "device-name-or-id", managedobjects.DeleteOptions{})

	assert.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)

	// The request URL should contain the resolved parameter
	// (In this mock test, it just passes through)
	assert.True(t, strings.Contains(prepared.Request.URL.Path, "device-name-or-id"))
}

func Test_DeferredExecution_ContextCheck(t *testing.T) {
	ctx := context.Background()

	// Default should be false
	assert.False(t, c8y_api.IsDeferredExecution(ctx))

	// Enable deferred execution
	ctx = c8y_api.WithDeferredExecution(ctx, true)
	assert.True(t, c8y_api.IsDeferredExecution(ctx))

	// Disable it
	ctx = c8y_api.WithDeferredExecution(ctx, false)
	assert.False(t, c8y_api.IsDeferredExecution(ctx))
}
