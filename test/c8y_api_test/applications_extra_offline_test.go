package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/applications"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Applications_ListByUser(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.Applications.ListByUser(ctx, applications.ListByUserOptions{
		Username: "admin",
	})
	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)
	_, err := op.ToSliceR(result)
	require.NoError(t, err)
}

func Test_Applications_Update(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	appName := "appupd" + testingutils.RandomString(6)
	createResult := client.Applications.Create(ctx, map[string]any{
		"name":        appName,
		"key":         appName + "-key",
		"type":        "HOSTED",
		"contextPath": appName,
	})
	require.NoError(t, createResult.Err)

	updResult := client.Applications.Update(ctx, createResult.Data.ID(), map[string]any{
		"availability": "MARKET",
	})
	require.NoError(t, updResult.Err)
	assert.Equal(t, 200, updResult.HTTPStatus)

	require.NoError(t, client.Applications.Delete(ctx, createResult.Data.ID(), applications.DeleteOptions{}).Err)
}

func Test_Applications_SubscribeUnsubscribe(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	appName := "appsub" + testingutils.RandomString(6)
	createResult := client.Applications.Create(ctx, map[string]any{
		"name":        appName,
		"key":         appName + "-key",
		"type":        "HOSTED",
		"contextPath": appName,
	})
	require.NoError(t, createResult.Err)
	defer client.Applications.Delete(ctx, createResult.Data.ID(), applications.DeleteOptions{})

	subResult := client.Applications.Subscribe(ctx, client.Auth.Tenant, createResult.Data.Self())
	require.NoError(t, subResult.Err)

	unsubResult := client.Applications.Unsubscribe(ctx, client.Auth.Tenant, createResult.Data.ID())
	require.NoError(t, unsubResult.Err)
}

func Test_Applications_ByID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	assert.Equal(t, "12345", client.Applications.ByID("12345"))
	assert.Equal(t, "name:cockpit", client.Applications.ByName("cockpit", ""))
	assert.Equal(t, "name:cockpit:HOSTED", client.Applications.ByName("cockpit", "HOSTED"))
}

func Test_Applications_GetByResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	// Use name:cockpit which is seeded
	result := client.Applications.Get(ctx, "name:cockpit")
	require.NoError(t, result.Err)
	assert.Equal(t, "cockpit", result.Data.Name())
}
