package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/bulkoperations"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func Test_BulkOperations_List_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.BulkOperations.List(ctx, bulkoperations.ListOptions{})

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
}

func Test_BulkOperations_List_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.BulkOperations.List(ctx, bulkoperations.ListOptions{})

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodGet, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/devicecontrol/bulkoperations")

	// Execute with dry run to confirm the full round-trip works
	result := prepared.Execute(api.WithDryRun(context.Background(), true))
	assert.False(t, result.IsDeferred())
	assert.NoError(t, result.Err)
}

func Test_BulkOperations_List_Pagination(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	opts := bulkoperations.ListOptions{}
	opts.PageSize = 5
	opts.WithTotalElements = true

	prepared := client.BulkOperations.List(ctx, opts)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Contains(t, prepared.Request.URL.RawQuery, "pageSize=5")
	assert.Contains(t, prepared.Request.URL.RawQuery, "withTotalElements=true")
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func Test_BulkOperations_Get_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.BulkOperations.Get(ctx, "11111")

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
}

func Test_BulkOperations_Get_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.BulkOperations.Get(ctx, "42")

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodGet, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/devicecontrol/bulkoperations/42")
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func Test_BulkOperations_Create_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	body := map[string]any{
		"groupId":      "100",
		"startDate":    time.Now().UTC().Format(time.RFC3339),
		"creationRamp": 0.5,
		"operationPrototype": map[string]any{
			"c8y_Restart": map[string]any{},
		},
	}

	result := client.BulkOperations.Create(ctx, body)

	assert.NoError(t, result.Err)
}

func Test_BulkOperations_Create_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	body := map[string]any{
		"groupId":      "100",
		"startDate":    time.Now().UTC().Format(time.RFC3339),
		"creationRamp": 0.5,
		"operationPrototype": map[string]any{
			"c8y_Restart": map[string]any{},
		},
	}

	prepared := client.BulkOperations.Create(ctx, body)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodPost, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/devicecontrol/bulkoperations")
	// Must not contain an ID segment (collection endpoint)
	assert.NotContains(t, prepared.Request.URL.Path, "/devicecontrol/bulkoperations/")
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func Test_BulkOperations_Update_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	body := map[string]any{
		"status": string(types.BulkOperationCompleted),
	}

	result := client.BulkOperations.Update(ctx, "42", body)

	assert.NoError(t, result.Err)
}

func Test_BulkOperations_Update_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	body := map[string]any{
		"creationRamp": 2.0,
	}

	prepared := client.BulkOperations.Update(ctx, "42", body)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodPut, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/devicecontrol/bulkoperations/42")
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func Test_BulkOperations_Delete_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.BulkOperations.Delete(ctx, "42")

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusNoContent, result.HTTPStatus)
}

func Test_BulkOperations_Delete_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.BulkOperations.Delete(ctx, "42")

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodDelete, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/devicecontrol/bulkoperations/42")
}
