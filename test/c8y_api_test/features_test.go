package api_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Features - List
// ---------------------------------------------------------------------------

func Test_Features_List_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), false)

	result := client.Features.List(ctx)

	for item := range op.Iter(result) {
		slog.Info("feature", "key", item.Key(), "phase", item.Phase(), "active", item.Active())
	}

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
}

func Test_Features_List_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.Features.List(ctx)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodGet, prepared.Request.Method)
	assert.Equal(t, "/features", prepared.Request.URL.Path)

	result := prepared.Execute(api.WithDryRun(context.Background(), true))
	assert.False(t, result.IsDeferred())
	assert.NoError(t, result.Err)
}

// ---------------------------------------------------------------------------
// Features - Get
// ---------------------------------------------------------------------------

func Test_Features_Get_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.Features.Get(ctx, "my-feature")

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
}

func Test_Features_Get_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.Features.Get(ctx, "my-feature")

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodGet, prepared.Request.Method)
	assert.Equal(t, "/features/my-feature", prepared.Request.URL.Path)
}

// ---------------------------------------------------------------------------
// Features - Enable / Disable
// ---------------------------------------------------------------------------

func Test_Features_Enable_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.Features.Enable(ctx, "my-feature")

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodPut, prepared.Request.Method)
	assert.Equal(t, "/features/my-feature", prepared.Request.URL.Path)
}

func Test_Features_Disable_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.Features.Disable(ctx, "my-feature")

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodPut, prepared.Request.Method)
	assert.Equal(t, "/features/my-feature", prepared.Request.URL.Path)
}

// ---------------------------------------------------------------------------
// Features - Delete
// ---------------------------------------------------------------------------

func Test_Features_Delete_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.Features.Delete(ctx, "my-feature")

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusNoContent, result.HTTPStatus)
}

func Test_Features_Delete_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.Features.Delete(ctx, "my-feature")

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodDelete, prepared.Request.Method)
	assert.Equal(t, "/features/my-feature", prepared.Request.URL.Path)
}

// ---------------------------------------------------------------------------
// TenantOverrides - List
// ---------------------------------------------------------------------------

func Test_Features_Tenants_List_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.Features.Tenants.List(ctx, "my-feature")

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
}

func Test_Features_Tenants_List_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.Features.Tenants.List(ctx, "my-feature")

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodGet, prepared.Request.Method)
	assert.Equal(t, "/features/my-feature/by-tenant", prepared.Request.URL.Path)

	result := prepared.Execute(api.WithDryRun(context.Background(), true))
	assert.False(t, result.IsDeferred())
	assert.NoError(t, result.Err)
}

// ---------------------------------------------------------------------------
// TenantOverrides - Enable / Disable (current tenant)
// ---------------------------------------------------------------------------

func Test_Features_Tenants_Enable_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.Features.Tenants.Enable(ctx, "my-feature")

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodPut, prepared.Request.Method)
	assert.Equal(t, "/features/my-feature/by-tenant", prepared.Request.URL.Path)
}

func Test_Features_Tenants_Disable_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.Features.Tenants.Disable(ctx, "my-feature")

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodPut, prepared.Request.Method)
	assert.Equal(t, "/features/my-feature/by-tenant", prepared.Request.URL.Path)
}

// ---------------------------------------------------------------------------
// TenantOverrides - Delete (current tenant)
// ---------------------------------------------------------------------------

func Test_Features_Tenants_Delete_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.Features.Tenants.Delete(ctx, "my-feature")

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusNoContent, result.HTTPStatus)
}

func Test_Features_Tenants_Delete_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.Features.Tenants.Delete(ctx, "my-feature")

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodDelete, prepared.Request.Method)
	assert.Equal(t, "/features/my-feature/by-tenant", prepared.Request.URL.Path)
}

// ---------------------------------------------------------------------------
// TenantOverrides - EnableForTenant / DisableForTenant
// ---------------------------------------------------------------------------

func Test_Features_Tenants_EnableForTenant_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.Features.Tenants.EnableForTenant(ctx, "my-feature", "t07007007")

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodPut, prepared.Request.Method)
	assert.Equal(t, "/features/my-feature/by-tenant/t07007007", prepared.Request.URL.Path)
}

func Test_Features_Tenants_DisableForTenant_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.Features.Tenants.DisableForTenant(ctx, "my-feature", "t07007007")

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodPut, prepared.Request.Method)
	assert.Equal(t, "/features/my-feature/by-tenant/t07007007", prepared.Request.URL.Path)
}

// ---------------------------------------------------------------------------
// TenantOverrides - DeleteForTenant
// ---------------------------------------------------------------------------

func Test_Features_Tenants_DeleteForTenant_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.Features.Tenants.DeleteForTenant(ctx, "my-feature", "t07007007")

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusNoContent, result.HTTPStatus)
}

func Test_Features_Tenants_DeleteForTenant_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.Features.Tenants.DeleteForTenant(ctx, "my-feature", "t07007007")

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodDelete, prepared.Request.Method)
	assert.Equal(t, "/features/my-feature/by-tenant/t07007007", prepared.Request.URL.Path)
}
