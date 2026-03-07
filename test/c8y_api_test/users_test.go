package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/users"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_Users(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)
	collection := client.Users.List(context.Background(), users.ListOptions{})
	assert.NoError(t, collection.Err)
	assert.Greater(t, collection.Data.Length(), 0)

	if collection.Data.Length() > 0 {
		userID := ""
		for user := range jsondoc.DecodeIter[model.User](collection.Data.Iter()) {
			userID = user.ID
			break
		}
		user := client.Users.Get(context.Background(), users.GetOptions{
			ID: userID,
		})
		assert.NoError(t, user.Err)
		assert.Equal(t, userID, user.Data.ID())
	}
}

func TestUserService_List(t *testing.T) {
	client := testcore.CreateTestClient(t)
	// list
	collection := client.Users.List(
		context.Background(),
		users.ListOptions{
			PaginationOptions: pagination.PaginationOptions{
				PageSize: 100,
			},
		},
	)
	assert.NoError(t, collection.Err)
	assert.Greater(t, collection.Data.Length(), 0, "At least 1 user should be present")

	// get by username
	// TODO: add a nicer way to get the first item in the array of results
	userName := ""
	for user := range jsondoc.DecodeIter[model.User](collection.Data.Iter()) {
		userName = user.Username
		break
	}

	user := client.Users.GetByUsername(context.Background(), users.GetByUsernameOptions{
		Username: userName,
	})
	assert.NoError(t, user.Err)
	assert.Equal(t, userName, user.Data.UserName())
}

func Test_GetUserTFA(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Use the currently authenticated user — always exists and accessible
	result := client.Users.GetTFA(ctx, users.GetTFAOptions{
		ID:     client.Auth.Username,
		Tenant: client.Auth.Tenant,
	})

	assert.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)
	// Strategy is only set when TFA is active; just verify the field is a known value when present
	strategy := result.Data.Strategy()
	if strategy != "" {
		assert.Contains(t, []string{"SMS", "TOTP"}, strategy)
	}
}
