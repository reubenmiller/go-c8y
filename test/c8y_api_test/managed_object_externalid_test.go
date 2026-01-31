package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/identity"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_ManagedObjectGetOrCreateByExternalID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	externalID := testingutils.RandomString(16)
	deviceName := testingutils.RandomString(16)

	// First call - should create both managed object and identity
	result1 := client.ManagedObjects.GetOrCreateByExternalID(ctx, managedobjects.GetOrCreateByExternalIDOptions{
		ExternalID:     externalID,
		ExternalIDType: "c8y_Serial",
		Body: map[string]any{
			"name":         deviceName,
			"type":         "c8y_Device",
			"c8y_IsDevice": map[string]any{},
		},
	})

	assert.NoError(t, result1.Err)
	assert.Equal(t, "Created", string(result1.Status))
	assert.Equal(t, false, result1.Meta["found"])
	assert.Equal(t, true, result1.Meta["identityAssigned"])
	assert.Equal(t, externalID, result1.Meta["externalID"])
	assert.Equal(t, deviceName, result1.Data.Name())
	assert.NotEmpty(t, result1.Data.ID())

	deviceID := result1.Data.ID()

	// Second call - should find existing managed object
	result2 := client.ManagedObjects.GetOrCreateByExternalID(ctx, managedobjects.GetOrCreateByExternalIDOptions{
		ExternalID:     externalID,
		ExternalIDType: "c8y_Serial",
		Body: map[string]any{
			"name":         "ShouldNotBeCreated",
			"type":         "c8y_Device",
			"c8y_IsDevice": map[string]any{},
		},
	})

	assert.NoError(t, result2.Err)
	assert.Equal(t, "OK", string(result2.Status))
	assert.Equal(t, true, result2.Meta["found"])
	assert.Equal(t, deviceID, result2.Data.ID())
	assert.Equal(t, deviceName, result2.Data.Name()) // Should have original name

	// Verify we can get it via identity
	identResult := client.Identity.Get(ctx, identity.IdentityOptions{
		ExternalID: externalID,
		Type:       "c8y_Serial",
	})
	assert.NoError(t, identResult.Err)
	assert.Equal(t, externalID, identResult.Data.ExternalID())
	assert.Equal(t, deviceID, identResult.Data.ManagedObjectID())

	// Cleanup
	deleteIdentity := client.Identity.Delete(ctx, identity.IdentityOptions{
		ExternalID: externalID,
		Type:       "c8y_Serial",
	})
	assert.NoError(t, deleteIdentity.Err)

	deleteMO := client.ManagedObjects.Delete(ctx, deviceID, managedobjects.DeleteOptions{})
	assert.NoError(t, deleteMO.Err)
}

func Test_ManagedObjectGetOrCreateByExternalID_DefaultType(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	externalID := testingutils.RandomString(16)

	// Test with default type (should use c8y_Serial)
	result := client.ManagedObjects.GetOrCreateByExternalID(ctx, managedobjects.GetOrCreateByExternalIDOptions{
		ExternalID: externalID,
		// ExternalIDType not specified - should default to c8y_Serial
		Body: map[string]any{
			"name":         "DefaultTypeTest",
			"type":         "c8y_Device",
			"c8y_IsDevice": map[string]any{},
		},
	})

	assert.NoError(t, result.Err)
	assert.Equal(t, "Created", string(result.Status))
	assert.Equal(t, "c8y_Serial", result.Meta["externalIDType"])

	// Verify default type was used
	identResult := client.Identity.Get(ctx, identity.IdentityOptions{
		ExternalID: externalID,
		Type:       "c8y_Serial",
	})
	assert.NoError(t, identResult.Err)

	// Cleanup
	client.Identity.Delete(ctx, identity.IdentityOptions{
		ExternalID: externalID,
		Type:       "c8y_Serial",
	})
	client.ManagedObjects.Delete(ctx, result.Data.ID(), managedobjects.DeleteOptions{})
}
