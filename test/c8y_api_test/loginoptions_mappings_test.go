package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/loginoptions"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_LoginOptionAccessMappings exercises the full access-mappings CRUD surface against
// the offline fake server (including the destructive create/update/delete operations).
func Test_LoginOptionAccessMappings(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("destructive auth-config writes: offline simulated backend only")
	}
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	const typeOrID = "OAUTH2"

	// Create
	created := client.LoginOptions.AccessMappings.Create(ctx, typeOrID, map[string]any{
		"when":             map[string]any{"operator": "AND"},
		"thenGroups":       []int{1},
		"thenApplications": []int{2},
	})
	require.NoError(t, created.Err)
	id := created.Data.ID()
	require.NotEmpty(t, id)

	// Get
	got := client.LoginOptions.AccessMappings.Get(ctx, typeOrID, id)
	require.NoError(t, got.Err)
	assert.Equal(t, id, got.Data.ID())

	// List
	list := client.LoginOptions.AccessMappings.List(ctx, typeOrID)
	require.NoError(t, list.Err)
	assert.GreaterOrEqual(t, list.Data.Length(), 1)

	// Update
	updated := client.LoginOptions.AccessMappings.Update(ctx, typeOrID, id, map[string]any{
		"thenGroups": []int{1, 3},
	})
	require.NoError(t, updated.Err)

	// Delete
	del := client.LoginOptions.AccessMappings.Delete(ctx, typeOrID, id)
	require.NoError(t, del.Err)

	// Confirm gone
	missing := client.LoginOptions.AccessMappings.Get(ctx, typeOrID, id)
	assert.Error(t, missing.Err)
}

// Test_LoginOptionInventoryAccessMappings exercises the inventory-access-mappings CRUD
// surface against the offline fake server.
func Test_LoginOptionInventoryAccessMappings(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("destructive auth-config writes: offline simulated backend only")
	}
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	const typeOrID = "OAUTH2"

	created := client.LoginOptions.InventoryAccessMappings.Create(ctx, typeOrID, map[string]any{
		"when":               map[string]any{"operator": "AND"},
		"thenInventoryRoles": []map[string]any{{"managedObject": "358104", "roleIds": []int{1}}},
	})
	require.NoError(t, created.Err)
	id := created.Data.ID()
	require.NotEmpty(t, id)

	got := client.LoginOptions.InventoryAccessMappings.Get(ctx, typeOrID, id)
	require.NoError(t, got.Err)
	assert.Equal(t, id, got.Data.ID())

	list := client.LoginOptions.InventoryAccessMappings.List(ctx, typeOrID)
	require.NoError(t, list.Err)
	assert.GreaterOrEqual(t, list.Data.Length(), 1)

	del := client.LoginOptions.InventoryAccessMappings.Delete(ctx, typeOrID, id)
	require.NoError(t, del.Err)
}

// Test_LoginOptionRestrict exercises PUT .../{typeOrId}/restrict.
func Test_LoginOptionRestrict(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("destructive auth-config write: offline simulated backend only")
	}
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.LoginOptions.Restrict(ctx, "OAUTH2", loginoptions.RestrictOptions{
		OnlyManagementTenantAccess: true,
	})
	require.NoError(t, result.Err)
	assert.True(t, result.Data.Get("onlyManagementTenantAccess").Bool())
}
