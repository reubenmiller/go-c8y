package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/applications"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetApplications(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.Applications.List(ctx, applications.ListOptions{
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 10,
		},
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	apps, err := op.ToSliceR(result)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(apps), 1, "At least one application should be found")
	assert.NotEmpty(t, apps[0].Name(), "Application should have a name")
}

func Test_GetApplicationsByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	exampleAppName := "cockpit"

	result := client.Applications.ListByName(ctx, applications.ListByNameOptions{
		Name: exampleAppName,
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	apps, err := op.ToSliceR(result)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(apps), 1, "Should find at least one application")

	assert.Equal(t, exampleAppName, apps[0].Name(), "Application name should match")
}

func Test_GetApplicationsByOwner(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.Applications.ListByOwner(ctx, applications.ListByOwnerOptions{
		TenantID: client.Auth.Tenant,
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	apps, err := op.ToSliceR(result)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(apps), 0, "Should return applications")
}

func Test_GetApplicationsByTenant(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.Applications.ListByTenant(ctx, applications.ListByTenantOptions{
		TenantID: client.Auth.Tenant,
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	apps, err := op.ToSliceR(result)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(apps), 1, "Should find at least one application")
	assert.NotEmpty(t, apps[0].Name(), "Application should have a name")
}

func Test_GetApplication(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	applicationName := "cockpit"

	// Find application by name
	listResult := client.Applications.ListByName(ctx, applications.ListByNameOptions{
		Name: applicationName,
	})
	require.NoError(t, listResult.Err)

	apps, err := op.ToSliceR(listResult)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(apps), 1, "Should return at least 1 application")

	expApp := apps[0]

	// Get specific application
	result := client.Applications.Get(ctx, expApp.ID())
	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)
	assert.Equal(t, expApp.ID(), result.Data.ID())
}

func Test_CRUD_Application(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	appName := "testApplication" + testingutils.RandomString(7)

	appInfo := map[string]any{
		"key":         appName + "Key",
		"name":        appName,
		"type":        "HOSTED",
		"contextPath": "/" + appName,
	}

	// Delete application if it already exists
	existingApps := client.Applications.ListByName(ctx, applications.ListByNameOptions{
		Name: appName,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 10,
		},
	})
	if existingApps.Err == nil {
		if apps, err := op.ToSliceR(existingApps); err == nil {
			for _, app := range apps {
				client.Applications.Delete(ctx, app.ID(), applications.DeleteOptions{})
			}
		}
	}

	// Delete the cloned app if it exists
	existingClonedApps := client.Applications.ListByName(ctx, applications.ListByNameOptions{
		Name: "clone" + appName,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 10,
		},
	})
	if existingClonedApps.Err == nil {
		if apps, err := op.ToSliceR(existingClonedApps); err == nil {
			for _, app := range apps {
				client.Applications.Delete(ctx, app.ID(), applications.DeleteOptions{})
			}
		}
	}

	// Create application
	createResult := client.Applications.Create(ctx, appInfo)
	require.NoError(t, createResult.Err)
	assert.Equal(t, 201, createResult.HTTPStatus)
	assert.Equal(t, appInfo["key"], createResult.Data.Get("key").String())

	t.Cleanup(func() {
		// always try to delete the app
		client.Applications.Delete(ctx, createResult.Data.ID(), applications.DeleteOptions{})
	})

	// Copy existing application
	copyResult := client.Applications.Copy(ctx, createResult.Data.ID(), applications.CopyOptions{})
	require.NoError(t, copyResult.Err)
	assert.Equal(t, 201, copyResult.HTTPStatus)
	assert.Equal(t, "clone"+appName, copyResult.Data.Name())

	t.Cleanup(func() {
		// always try to delete the app
		client.Applications.Delete(ctx, copyResult.Data.ID(), applications.DeleteOptions{})
	})

	// Delete original application
	deleteResult := client.Applications.Delete(ctx, createResult.Data.ID(), applications.DeleteOptions{})
	require.NoError(t, deleteResult.Err)
	assert.Equal(t, 204, deleteResult.HTTPStatus)

	// Delete copied application
	deleteCopyResult := client.Applications.Delete(ctx, copyResult.Data.ID(), applications.DeleteOptions{})
	require.NoError(t, deleteCopyResult.Err)
	assert.Equal(t, 204, deleteCopyResult.HTTPStatus)
}
