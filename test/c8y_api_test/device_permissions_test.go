package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/users/devicepermissions"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Device Permissions - Get
// ---------------------------------------------------------------------------

func Test_DevicePermissions_Get_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.DevicePermissions.GetDevicePermissions(ctx, devicepermissions.GetDevicePermissionsOptions{
		ManagedObjectID: "12345",
	})

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
}

func Test_DevicePermissions_Get_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.DevicePermissions.GetDevicePermissions(ctx, devicepermissions.GetDevicePermissionsOptions{
		ManagedObjectID: "12345",
	})

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodGet, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/user/devicePermissions/12345")
}

// ---------------------------------------------------------------------------
// Device Permissions - Update
// ---------------------------------------------------------------------------

func Test_DevicePermissions_Update_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	// UpdatedDevicePermissions shape: users and groups with per-device permission maps.
	body := map[string]any{
		"users": []map[string]any{{
			"userName": "jdoe",
			"devicePermissions": map[string]any{
				"12345": []string{"MANAGED_OBJECT:*:ADMIN"},
			},
		}},
		"groups": []map[string]any{},
	}

	result := client.DevicePermissions.UpdateDevicePermissions(ctx, devicepermissions.GetDevicePermissionsOptions{
		ManagedObjectID: "12345",
	}, body)

	assert.NoError(t, result.Err)
}

func Test_DevicePermissions_Update_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	body := map[string]any{
		"users": []map[string]any{{
			"userName": "jdoe",
			"devicePermissions": map[string]any{
				"12345": []string{"READ"},
			},
		}},
		"groups": []map[string]any{},
	}

	prepared := client.DevicePermissions.UpdateDevicePermissions(ctx, devicepermissions.GetDevicePermissionsOptions{
		ManagedObjectID: "12345",
	}, body)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodPut, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/user/devicePermissions/12345")
}

// ---------------------------------------------------------------------------
// Inventory Role Assignments - List
// ---------------------------------------------------------------------------

func Test_DevicePermissions_ListAssignments_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.DevicePermissions.ListInventoryRoleAssignments(ctx, devicepermissions.ListInventoryRoleAssignmentOptions{
		UserScopedOptions: devicepermissions.UserScopedOptions{UserID: "testuser"},
	})

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
}

func Test_DevicePermissions_ListAssignments_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.DevicePermissions.ListInventoryRoleAssignments(ctx, devicepermissions.ListInventoryRoleAssignmentOptions{
		UserScopedOptions: devicepermissions.UserScopedOptions{UserID: "testuser"},
	})

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodGet, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/users/testuser/roles/inventory")
	assert.NotContains(t, prepared.Request.URL.Path, "/roles/inventory/")

	result := prepared.Execute(api.WithDryRun(context.Background(), true))
	assert.False(t, result.IsDeferred())
	assert.NoError(t, result.Err)
}

func Test_DevicePermissions_ListAssignments_Pagination(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	opts := devicepermissions.ListInventoryRoleAssignmentOptions{
		UserScopedOptions: devicepermissions.UserScopedOptions{UserID: "testuser"},
	}
	opts.PageSize = 5
	opts.WithTotalElements = true

	prepared := client.DevicePermissions.ListInventoryRoleAssignments(ctx, opts)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Contains(t, prepared.Request.URL.RawQuery, "pageSize=5")
	assert.Contains(t, prepared.Request.URL.RawQuery, "withTotalElements=true")
}

// ---------------------------------------------------------------------------
// Inventory Role Assignments - Assign
// ---------------------------------------------------------------------------

func Test_DevicePermissions_AssignRole_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	// Per OAS, managedObject is a plain string (managed-object ID), not an object.
	body := map[string]any{
		"managedObject": "12345",
		"roles":         []map[string]any{{"id": 1}},
	}

	result := client.DevicePermissions.AssignInventoryRole(ctx, devicepermissions.UserScopedOptions{
		UserID: "testuser",
	}, body)

	assert.NoError(t, result.Err)
}

func Test_DevicePermissions_AssignRole_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	body := map[string]any{
		"managedObject": "12345",
		"roles":         []map[string]any{{"id": 1}},
	}

	prepared := client.DevicePermissions.AssignInventoryRole(ctx, devicepermissions.UserScopedOptions{
		UserID: "testuser",
	}, body)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodPost, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/users/testuser/roles/inventory")
	assert.NotContains(t, prepared.Request.URL.Path, "/roles/inventory/")
}

// ---------------------------------------------------------------------------
// Inventory Role Assignment - Get single
// ---------------------------------------------------------------------------

func Test_DevicePermissions_GetAssignment_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.DevicePermissions.GetInventoryRoleAssignment(ctx, devicepermissions.GetInventoryRoleAssignmentOptions{
		UserScopedOptions: devicepermissions.UserScopedOptions{UserID: "testuser"},
		ID:                1,
	})

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
}

func Test_DevicePermissions_GetAssignment_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.DevicePermissions.GetInventoryRoleAssignment(ctx, devicepermissions.GetInventoryRoleAssignmentOptions{
		UserScopedOptions: devicepermissions.UserScopedOptions{UserID: "testuser"},
		ID:                42,
	})

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodGet, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/users/testuser/roles/inventory/42")
}

// ---------------------------------------------------------------------------
// Inventory Role Assignment - Update
// ---------------------------------------------------------------------------

func Test_DevicePermissions_UpdateAssignment_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	body := map[string]any{
		"roles": []map[string]any{{"id": 2}},
	}

	result := client.DevicePermissions.UpdateInventoryRoleAssignment(ctx, devicepermissions.GetInventoryRoleAssignmentOptions{
		UserScopedOptions: devicepermissions.UserScopedOptions{UserID: "testuser"},
		ID:                42,
	}, body)

	assert.NoError(t, result.Err)
}

func Test_DevicePermissions_UpdateAssignment_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	body := map[string]any{
		"roles": []map[string]any{{"id": 2}},
	}

	prepared := client.DevicePermissions.UpdateInventoryRoleAssignment(ctx, devicepermissions.GetInventoryRoleAssignmentOptions{
		UserScopedOptions: devicepermissions.UserScopedOptions{UserID: "testuser"},
		ID:                42,
	}, body)

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodPut, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/users/testuser/roles/inventory/42")
}

// ---------------------------------------------------------------------------
// Inventory Role Assignment - Delete
// ---------------------------------------------------------------------------

func Test_DevicePermissions_DeleteAssignment_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)

	result := client.DevicePermissions.DeleteInventoryRoleAssignment(ctx, devicepermissions.GetInventoryRoleAssignmentOptions{
		UserScopedOptions: devicepermissions.UserScopedOptions{UserID: "testuser"},
		ID:                42,
	})

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusNoContent, result.HTTPStatus)
}

func Test_DevicePermissions_DeleteAssignment_RequestInspection(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	prepared := client.DevicePermissions.DeleteInventoryRoleAssignment(ctx, devicepermissions.GetInventoryRoleAssignmentOptions{
		UserScopedOptions: devicepermissions.UserScopedOptions{UserID: "testuser"},
		ID:                42,
	})

	require.True(t, prepared.IsDeferred())
	require.NotNil(t, prepared.Request)
	assert.Equal(t, http.MethodDelete, prepared.Request.Method)
	assert.Contains(t, prepared.Request.URL.Path, "/users/testuser/roles/inventory/42")
}
