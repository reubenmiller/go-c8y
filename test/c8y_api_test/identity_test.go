package c8y_api_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/identity"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_IdentityCRUD(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	slog.Info("Setup")
	mo, err := testcore.CreateManagedObject(t, client)
	assert.NoError(t, err)

	// Create
	slog.Info("Create")
	externalID := testingutils.RandomString(16)
	ident, err := client.Identity.Create(context.Background(), mo.ID, identity.IdentityOptions{
		ExternalID: externalID,
	})
	assert.NoError(t, err)
	assert.Equal(t, ident.ExternalID, externalID)
	assert.Equal(t, ident.ManagedObject.ID, mo.ID)

	// List
	slog.Info("List")
	idents, err := client.Identity.List(context.Background(), mo.ID)
	assert.NoError(t, err)
	assert.Len(t, idents.Identities, 1)
	assert.Equal(t, idents.Identities[0].ExternalID, externalID)
	assert.Equal(t, idents.Identities[0].Type, identity.DefaultType)
	assert.Equal(t, idents.Identities[0].ManagedObject.ID, mo.ID)
	assert.NotEmpty(t, idents.Identities[0].ManagedObject.Self)

	// Get
	slog.Info("Get")
	ident, err = client.Identity.Get(context.Background(), identity.IdentityOptions{
		ExternalID: externalID,
	})
	assert.NoError(t, err)
	assert.Equal(t, ident.ExternalID, externalID)
	assert.Equal(t, ident.ManagedObject.ID, mo.ID)

	// Delete
	slog.Info("Delete")
	err = client.Identity.Delete(context.Background(), identity.IdentityOptions{
		ExternalID: externalID,
	})
	assert.NoError(t, err)
}
