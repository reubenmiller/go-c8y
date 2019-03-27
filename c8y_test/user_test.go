package c8y_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

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

func TestUserService_CRUD(t *testing.T) {
	client := createTestClient()

	userInput := c8y.NewUser("myciuser01", "myciuser01@no-reply.org", "0d18dksd81j30d*64fl65")
	userInput.
		SetFirstName("User01").
		SetLastName("CI")

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
	// Update user
	updatedUser, resp, err := client.User.Update(
		context.Background(),
		user.ID,
		&c8y.User{
			FirstName: "Alfred",
			LastName:  "Peabody",
			Phone:     "+61 7 1234 5678", // Only accepts landlines!
			// Phone: "+61 152 456 679",
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, "Alfred", updatedUser.FirstName)
	testingutils.Equals(t, "Peabody", updatedUser.LastName)
	testingutils.Equals(t, "+61 7 1234 5678", updatedUser.Phone)

	//
	// Delete user
	resp, err = client.User.Delete(
		context.Background(),
		user.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode)
}

func TestUserService_GetGroupByName(t *testing.T) {
	client := createTestClient()

	group, resp, err := client.User.GetGroupByName(
		context.Background(),
		"admins",
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, "admins", group.Name)
}

func TestUserService_UpdateCurrentUser(t *testing.T) {
	client := createTestClient()

	randomFirstName := fmt.Sprintf("testUser-%d", time.Now().Unix())
	user, resp, err := client.User.UpdateCurrentUser(
		context.Background(),
		&c8y.User{
			FirstName: randomFirstName,
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, client.Username, user.Username)
	testingutils.Equals(t, randomFirstName, user.FirstName)
}

func TestUserService_AddUserToGroup(t *testing.T) {
	client := createTestClient()

	// Get user
	currentUser, _, err := client.User.GetUserByUsername(
		context.Background(),
		client.Username,
	)
	testingutils.Ok(t, err)

	//
	// Get group
	group, resp, err := client.User.GetGroupByName(
		context.Background(),
		"Cockpit User",
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Assert(t, group.ID > 0, "ID should be greater than 0")

	//
	// Add user to group
	userRef, resp, err := client.User.AddUserToGroup(
		context.Background(),
		currentUser,
		group.ID,
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode)
	testingutils.Assert(t, userRef.Self != "", "Self link should not be empty")

	// Get the users in the group
	userReferences, resp, err := client.User.GetUsersByGroup(
		context.Background(),
		group.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Assert(t, len(userReferences.References) > 0, "Should be at least 1 user")

	//
	// Remove user from group
	resp, err = client.User.RemoveUserFromGroup(
		context.Background(),
		currentUser.Username,
		group.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode)
}

func TestUserService_GetUsersByGroup(t *testing.T) {
	client := createTestClient()

	// Get current user
	currentUser, resp, err := client.User.GetUserByUsername(
		context.Background(),
		client.Username,
	)
	testingutils.Ok(t, err)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)

	// Create temp group
	group, resp, err := client.User.CreateGroup(
		context.Background(),
		&c8y.Group{
			Name: "CustomCIGroup",
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode)
	testingutils.Equals(t, "CustomCIGroup", group.Name)

	// Add user to temp group
	_, resp, err = client.User.AddUserToGroup(
		context.Background(),
		currentUser,
		group.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode)

	// Get users in group
	userReferences, resp, err := client.User.GetUsersByGroup(
		context.Background(),
		group.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Assert(t, len(userReferences.References) > 0, "Should be at least one user")

	// Remove temp group
	resp, err = client.User.DeleteGroup(
		context.Background(),
		group.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode)
}
