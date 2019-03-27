package c8y_test

import (
	"context"
	"net/http"
	"testing"

	c8y "github.com/reubenmiller/go-c8y"
	"github.com/reubenmiller/go-c8y/c8y_test/testingutils"
)

func TestUserService_GetUsers(t *testing.T) {
	client := createTestClient()
	users, resp, err := client.User.GetUsers(
		context.Background(),
		&c8y.UserOptions{
			PaginationOptions: c8y.PaginationOptions{
				PageSize: 100,
			},
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Assert(t, len(users.Users) > 0, "At least 1 user should be present")
}

func TestUserService_GetUser(t *testing.T) {
	client := createTestClient()

	user, resp, err := client.User.GetUser(
		context.Background(),
		client.Username,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, user.ID, client.Username)
}

func TestUserService_GetUserByUsername(t *testing.T) {
	client := createTestClient()

	user, resp, err := client.User.GetUserByUsername(
		context.Background(),
		client.Username,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, user.ID, client.Username)
}

func TestUserService_Create(t *testing.T) {
	client := createTestClient()

	userInput := &c8y.User{
		Username:  "myciuser01",
		Email:     "myciuser01@no-reply.org",
		FirstName: "User01",
		LastName:  "CI",
		Password:  "0d18dksd81j30d*64fl65",
		Enabled:   true,
	}

	user, resp, err := client.User.Create(
		context.Background(),
		userInput,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode)
	testingutils.Equals(t, userInput.Username, user.ID) // Id is the same as the username
	testingutils.Equals(t, userInput.Username, user.Username)
	testingutils.Equals(t, userInput.FirstName, user.FirstName)
	testingutils.Equals(t, userInput.LastName, user.LastName)

	//
	// Delete ther user
	resp, err = client.User.Delete(
		context.Background(),
		user.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode)
}
