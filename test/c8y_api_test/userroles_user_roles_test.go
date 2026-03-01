package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/userroles/users"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_UserRoles_Users_List_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.UserRoles.Users.List(ctx, users.ListOptions{
		TenantID: "t12345",
		UserID:   "user1",
	})

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
}

func Test_UserRoles_Users_List_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.UserRoles.Users.List(ctx, users.ListOptions{
		TenantID: "t99999",
		UserID:   "myuser",
	})

	require.NotNil(t, result.Request)
	assert.Equal(t, http.MethodGet, result.Request.Method)
	assert.Contains(t, result.Request.URL.Path, "/user/t99999/users/myuser/roles")
	assert.NotEmpty(t, result.Request.Header.Get("Accept"))
}

func Test_UserRoles_Users_List_Pagination(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	opts := users.ListOptions{
		TenantID: "t12345",
		UserID:   "user1",
	}
	opts.PageSize = 5
	opts.WithTotalElements = true

	prepared := client.UserRoles.Users.List(ctx, opts)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Contains(t, prepared.Request.URL.RawQuery, "pageSize=5")
	assert.Contains(t, prepared.Request.URL.RawQuery, "withTotalElements=true")
}
