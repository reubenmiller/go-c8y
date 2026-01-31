package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/identity"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_IdentityCRUD(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create a managed object first
	mo := testcore.CreateManagedObject(t, client)
	assert.NoError(t, mo.Err)

	id := mo.Data.ID()
	externalID := testingutils.RandomString(16)

	// Create identity
	createResult := client.Identity.Create(ctx, id, identity.IdentityOptions{
		ExternalID: externalID,
	})
	assert.NoError(t, createResult.Err)
	assert.Equal(t, "Created", string(createResult.Status))
	assert.Equal(t, externalID, createResult.Data.ExternalID())
	assert.Equal(t, id, createResult.Data.ManagedObjectID())

	// List identities
	listResult := client.Identity.List(ctx, id)
	assert.NoError(t, listResult.Err)
	assert.Equal(t, "OK", string(listResult.Status))

	var foundIdentity bool
	for doc := range listResult.Data.Iter() {
		ident := jsonmodels.NewIdentity(doc.Bytes())
		if ident.ExternalID() == externalID {
			foundIdentity = true
			assert.Equal(t, identity.DefaultType, ident.Type())
			assert.Equal(t, id, ident.ManagedObjectID())
			break
		}
	}
	assert.True(t, foundIdentity, "Created identity should be in list")

	// Get identity
	getResult := client.Identity.Get(ctx, identity.IdentityOptions{
		ExternalID: externalID,
	})
	assert.NoError(t, getResult.Err)
	assert.Equal(t, "OK", string(getResult.Status))
	assert.Equal(t, externalID, getResult.Data.ExternalID())
	assert.Equal(t, id, getResult.Data.ManagedObjectID())

	// Delete identity
	deleteResult := client.Identity.Delete(ctx, identity.IdentityOptions{
		ExternalID: externalID,
	})
	assert.NoError(t, deleteResult.Err)

	// Cleanup managed object
	delMO := client.ManagedObjects.Delete(ctx, id, managedobjects.DeleteOptions{})
	assert.NoError(t, delMO.Err)
}
