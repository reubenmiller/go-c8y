package api_test

import (
	"context"
	"io"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/currenttenant"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_TenantsList(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)
	item := client.Tenants.List(context.Background(), tenants.ListOptions{})
	assert.NoError(t, item.Err)
	assert.NotEmpty(t, item.Meta["self"])
}

func Test_GetCurrent(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	// Get Current
	currentTenant := client.Tenants.Current.Get(context.Background(), currenttenant.GetOptions{})
	assert.NoError(t, currentTenant.Err)
	assert.NotEmpty(t, currentTenant.Data.Name())
}

func Test_GetTenantTFA(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Empty tenantID defers to the current context tenant via middleware
	result := client.Tenants.GetTFA(ctx, "")

	assert.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)
	// The strategy field must be one of the documented enum values when set
	strategy := result.Data.Strategy()
	if strategy != "" {
		assert.Contains(t, []string{"SMS", "TOTP"}, strategy)
	}
}

func Test_UpdateTenantTFA(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Use dry run: the PUT is written to the request but never sent to the server
	ctx := api.WithDryRun(context.Background(), true)

	result := client.Tenants.UpdateTFA(ctx, "", map[string]any{
		"strategy": "TOTP",
	})

	require.NoError(t, result.Err)
	require.NotNil(t, result.Request, "Request should be captured in dry-run result")

	assert.Equal(t, "PUT", result.Request.Method)
	assert.Contains(t, result.Request.URL.Path, "/tfa")
	assert.Equal(t, "application/json", result.Request.Header.Get("Content-Type"))

	body, err := io.ReadAll(result.Request.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "TOTP")
}
