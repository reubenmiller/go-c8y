package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/loginoptions"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LoginOptions_Lifecycle(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	createRes := client.LoginOptions.Create(ctx, map[string]any{
		"type":                 "OAUTH2",
		"grantType":            "AUTHORIZATION_CODE",
		"userManagementSource": "INTERNAL",
		"visibleOnLoginPage":   true,
	})
	require.NoError(t, createRes.Err)
	id := createRes.Data.ID()
	require.NotEmpty(t, id)

	getRes := client.LoginOptions.Get(ctx, id)
	require.NoError(t, getRes.Err)
	assert.Equal(t, id, getRes.Data.ID())

	updRes := client.LoginOptions.Update(ctx, id, map[string]any{
		"visibleOnLoginPage": false,
	})
	require.NoError(t, updRes.Err)

	listRes := client.LoginOptions.List(ctx, loginoptions.ListOptions{})
	require.NoError(t, listRes.Err)

	itAll := client.LoginOptions.ListAll(ctx, loginoptions.ListOptions{})
	require.NoError(t, itAll.Err())

	noAuth := client.LoginOptions.ListNoAuth(ctx, loginoptions.ListOptions{})
	require.NoError(t, noAuth.Err)

	restrictRes := client.LoginOptions.Restrict(ctx, id, loginoptions.RestrictOptions{
		OnlyManagementTenantAccess: true,
	})
	require.NoError(t, restrictRes.Err)

	updateAccessRes := client.LoginOptions.UpdateAccess(ctx, loginoptions.UpdateAccessOptions{
		TypeOrId:     id,
		TargetTenant: "t12345",
	}, loginoptions.LoginOptionTenantAccess{OnlyManagementTenantAccess: false})
	require.NoError(t, updateAccessRes.Err)

	delRes := client.LoginOptions.Delete(ctx, id)
	require.NoError(t, delRes.Err)
}
