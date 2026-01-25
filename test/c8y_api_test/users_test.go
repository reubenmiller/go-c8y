package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/users"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_Users(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)
	collection, err := client.Users.List(context.Background(), users.ListOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, collection)

	if len(collection.Users) > 0 {
		user, err := client.Users.Get(context.Background(), users.Target{
			ID: collection.Users[0].ID,
		})
		assert.NoError(t, err)
		assert.Equal(t, user.ID, collection.Users[0].ID)
	}
}

func TestUserService_List(t *testing.T) {
	client := testcore.CreateTestClient(t)
	// list
	collection, err := client.Users.List(
		context.Background(),
		users.ListOptions{
			PaginationOptions: pagination.PaginationOptions{
				PageSize: 100,
			},
		},
	)
	assert.NoError(t, err)
	assert.Greater(t, len(collection.Users), 0, "At least 1 user should be present")

	// get by username
	user, err := client.Users.GetByUsername(context.Background(), users.GetByUsernameOptions{
		Username: collection.Users[0].Username,
	})
	assert.NoError(t, err)
	assert.Equal(t, user.Username, collection.Users[0].Username)
}
