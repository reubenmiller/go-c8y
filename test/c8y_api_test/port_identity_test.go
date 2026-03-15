package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/identity"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CreateIdentity(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	device := testcore.CreateDevice(t, client).Data

	identityName := "ext_" + testingutils.RandomString(16)

	// Create external identity
	result := client.Identity.Create(ctx, device.ID(), identity.IdentityOptions{
		Type:       "test_Type",
		ExternalID: identityName,
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 201, result.HTTPStatus)
	assert.Equal(t, device.ID(), result.Data.ManagedObjectID())
	assert.Equal(t, "test_Type", result.Data.Type())
	assert.Equal(t, identityName, result.Data.ExternalID())

	// Get identity object
	getResult := client.Identity.Get(ctx, identity.IdentityOptions{
		Type:       "test_Type",
		ExternalID: identityName,
	})

	require.NoError(t, getResult.Err)
	assert.Equal(t, 200, getResult.HTTPStatus)
	assert.Equal(t, device.ID(), getResult.Data.ManagedObjectID())

	moID := getResult.Data.Get("managedObject.id").String()
	assert.Equal(t, device.ID(), moID)
}

func Test_GetExternalID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.Identity.Get(ctx, identity.IdentityOptions{
		Type:       "NoExistentType",
		ExternalID: "Value123",
	})

	assert.Error(t, result.Err, "Error should not be nil")
	assert.Equal(t, 404, result.HTTPStatus)
	assert.Empty(t, result.Data.Type())
}

func Test_DeleteIdentity(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	device := testcore.CreateDevice(t, client).Data

	identityType := "testType"
	externalID := "ext" + testingutils.RandomString(16)

	// Create identity
	createResult := client.Identity.Create(ctx, device.ID(), identity.IdentityOptions{
		Type:       identityType,
		ExternalID: externalID,
	})

	require.NoError(t, createResult.Err)
	assert.Equal(t, 201, createResult.HTTPStatus)
	assert.Equal(t, identityType, createResult.Data.Type())
	assert.Equal(t, externalID, createResult.Data.ExternalID())

	// Remove identity
	deleteResult := client.Identity.Delete(ctx, identity.IdentityOptions{
		Type:       identityType,
		ExternalID: externalID,
	})

	require.NoError(t, deleteResult.Err)
	assert.Equal(t, 204, deleteResult.HTTPStatus)

	// Check that it was really deleted
	getResult := client.Identity.Get(ctx, identity.IdentityOptions{
		Type:       identityType,
		ExternalID: externalID,
	})

	assert.Error(t, getResult.Err, "Error should not be nil")
	assert.Equal(t, 404, getResult.HTTPStatus)
	assert.Empty(t, getResult.Data.Type())
}
