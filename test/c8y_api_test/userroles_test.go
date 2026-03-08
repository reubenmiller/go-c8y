package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/userroles"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_UserRoles(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.SetDebug(true)
	collection := client.UserRoles.List(context.Background(), userroles.ListOptions{})
	assert.NoError(t, collection.Err)
	assert.NotNil(t, collection)
	assert.Greater(t, collection.Data.Length(), 0)

	firstItem, err := collection.First()
	assert.NoError(t, err)

	role := client.UserRoles.Get(context.Background(), userroles.GetOption{
		Name: firstItem.Name(),
	})
	assert.NoError(t, role.Err)
	assert.Equal(t, role.Data.Name(), firstItem.Name())
}
