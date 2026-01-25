package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_CurrentUser(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	// get
	currentUser, err := client.Users.CurrentUser.Get(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, currentUser)
	assert.NotEmpty(t, currentUser.ID)

	// update - don't modify the current test user
	// updatedUser, err := client.Users.CurrentUser.Update(context.Background(), model.User{
	// 	CustomProperties: map[string]any{
	// 		"ci_testing": testingutils.RandomString(16),
	// 	},
	// })
	// assert.NoError(t, err)
	// assert.NotNil(t, updatedUser)
}
