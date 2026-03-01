package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	inventoryroles "github.com/reubenmiller/go-c8y/pkg/c8y/api/userroles/inventory"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func Test_InventoryRoles_List_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.InventoryRoles.List(ctx, inventoryroles.ListOptions{})

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
}

func Test_InventoryRoles_List_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.InventoryRoles.List(ctx, inventoryroles.ListOptions{})

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodGet, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/user/inventoryroles")
	assert.NotContains(t, prepared.Request.URL.Path, "/user/inventoryroles/")

	result := prepared.Execute(api.WithDryRun(context.Background(), true))
	assert.False(t, result.IsDeferred())
	assert.NoError(t, result.Err)
}

func Test_InventoryRoles_List_Pagination(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	opts := inventoryroles.ListOptions{}
	opts.PageSize = 5
	opts.WithTotalElements = true

	prepared := client.InventoryRoles.List(ctx, opts)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Contains(t, prepared.Request.URL.RawQuery, "pageSize=5")
	assert.Contains(t, prepared.Request.URL.RawQuery, "withTotalElements=true")
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func Test_InventoryRoles_Get_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.InventoryRoles.Get(ctx, 1)

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
}

func Test_InventoryRoles_Get_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.InventoryRoles.Get(ctx, 42)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodGet, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/user/inventoryroles/42")
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func Test_InventoryRoles_Create_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	body := map[string]any{
		"name":        "TestInventoryRole",
		"description": "A test inventory role",
	}

	result := client.InventoryRoles.Create(ctx, body)

	assert.NoError(t, result.Err)
}

func Test_InventoryRoles_Create_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	body := map[string]any{
		"name":        "TestInventoryRole",
		"description": "A test inventory role",
	}

	prepared := client.InventoryRoles.Create(ctx, body)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodPost, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/user/inventoryroles")
	assert.NotContains(t, prepared.Request.URL.Path, "/user/inventoryroles/")
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func Test_InventoryRoles_Update_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	body := map[string]any{
		"description": "Updated description",
	}

	result := client.InventoryRoles.Update(ctx, 42, body)

	assert.NoError(t, result.Err)
}

func Test_InventoryRoles_Update_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	body := map[string]any{
		"name": "UpdatedInventoryRole",
	}

	prepared := client.InventoryRoles.Update(ctx, 42, body)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodPut, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/user/inventoryroles/42")
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func Test_InventoryRoles_Delete_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.InventoryRoles.Delete(ctx, 42)

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusNoContent, result.HTTPStatus)
}

func Test_InventoryRoles_Delete_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.InventoryRoles.Delete(ctx, 42)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodDelete, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/user/inventoryroles/42")
}
