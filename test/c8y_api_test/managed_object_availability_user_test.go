package api_test

import (
	"context"
	"io"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Availability ---

func Test_ManagedObject_GetAvailability_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.ManagedObjects.GetAvailability(ctx, "12345")

	require.NoError(t, result.Err)
	require.NotNil(t, result.Request)
	assert.Equal(t, "GET", result.Request.Method)
	assert.Contains(t, result.Request.URL.Path, "/inventory/managedObjects/12345/availability")
}

func Test_ManagedObject_GetAvailability_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.ManagedObjects.GetAvailability(ctx, "99999")

	require.NotNil(t, result.Request)
	assert.Equal(t, "GET", result.Request.Method)
	assert.Contains(t, result.Request.URL.Path, "99999")
	assert.Contains(t, result.Request.URL.Path, "availability")
	assert.NotEmpty(t, result.Request.Header.Get("Accept"))
}

// --- User (owner) ---

func Test_ManagedObject_GetUser_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.ManagedObjects.GetUser(ctx, "12345")

	require.NoError(t, result.Err)
	require.NotNil(t, result.Request)
	assert.Equal(t, "GET", result.Request.Method)
	assert.Contains(t, result.Request.URL.Path, "/inventory/managedObjects/12345/user")
}

func Test_ManagedObject_GetUser_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.ManagedObjects.GetUser(ctx, "99999")

	require.NotNil(t, result.Request)
	assert.Equal(t, "GET", result.Request.Method)
	assert.Contains(t, result.Request.URL.Path, "99999")
	assert.Contains(t, result.Request.URL.Path, "/user")
}

func Test_ManagedObject_UpdateUser_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.ManagedObjects.UpdateUser(ctx, "12345", map[string]any{
		"enabled": false,
	})

	require.NoError(t, result.Err)
	require.NotNil(t, result.Request)
	assert.Equal(t, "PUT", result.Request.Method)
	assert.Contains(t, result.Request.URL.Path, "/inventory/managedObjects/12345/user")
	assert.Contains(t, result.Request.Header.Get("Content-Type"), "managedobjectuser")
}

func Test_ManagedObject_UpdateUser_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.ManagedObjects.UpdateUser(ctx, "12345", map[string]any{
		"enabled": true,
	})

	require.NotNil(t, result.Request)
	assert.Equal(t, "PUT", result.Request.Method)

	body, err := io.ReadAll(result.Request.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "enabled")
}
