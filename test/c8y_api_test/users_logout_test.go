package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Users_Logout_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)
	result := client.Users.Logout(ctx)
	assert.NoError(t, result.Err)
}

func Test_Users_Logout_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)
	prepared := client.Users.Logout(ctx)
	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodPost, prepared.Request.Method)
	assert.Equal(t, "/user/logout", prepared.Request.URL.Path)
	result := prepared.Execute(api.WithDryRun(context.Background(), true))
	assert.False(t, result.IsDeferred())
	assert.NoError(t, result.Err)
}

func Test_Users_LogoutAllUsers_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)
	result := client.Users.LogoutAllUsers(ctx, "t123")
	assert.NoError(t, result.Err)
}

func Test_Users_LogoutAllUsers_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)
	prepared := client.Users.LogoutAllUsers(ctx, "t123")
	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodPost, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/user/logout/t123/allUsers")
	result := prepared.Execute(api.WithDryRun(context.Background(), true))
	assert.False(t, result.IsDeferred())
	assert.NoError(t, result.Err)
}
