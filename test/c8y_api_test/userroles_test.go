package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/userroles"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_UserRoles(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)
	collection, err := client.UserRoles.List(context.Background(), userroles.ListOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, collection)
	assert.Greater(t, len(collection.Roles), 0)

	role, err := client.UserRoles.Get(context.Background(), userroles.GetOption{
		Name: collection.Roles[0].Name,
	})
	assert.NoError(t, err)
	assert.Equal(t, role.Name, collection.Roles[0].Name)
}
