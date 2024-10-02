package c8y_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
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
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Assert(t, len(users.Users) > 0, "At least 1 user should be present")
}

func TestUserService_GetUser(t *testing.T) {
	client := createTestClient()

	user, resp, err := client.User.GetUser(
		context.Background(),
		client.Username,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, user.ID, client.Username)
}

func TestUserService_GetUserByUsername(t *testing.T) {
	client := createTestClient()

	user, resp, err := client.User.GetUserByUsername(
		context.Background(),
		client.Username,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, user.ID, client.Username)
}

func TestUserService_CRUD(t *testing.T) {
	client := createTestClient()
	name := "myciuser" + testingutils.RandomString(7)
	password := testingutils.RandomPassword(32)
	userInput := c8y.NewUser(name, name+"@no-reply.org", password)
	userInput.
		SetFirstName("User01").
		SetLastName("CI")

	user, resp, err := client.User.Create(
		context.Background(),
		userInput,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
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
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
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
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())
}

func TestUserService_GetGroupByName(t *testing.T) {
	client := createTestClient()

	group, resp, err := client.User.GetGroupByName(
		context.Background(),
		"admins",
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, "admins", group.Name)

	//
	// Get group by id
	groupByID, resp, err := client.User.GetGroup(
		context.Background(),
		group.GetID(),
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, "admins", groupByID.Name)
}

func TestUserService_GetCurrentUser(t *testing.T) {
	client := createTestClient()

	user, resp, err := client.User.GetCurrentUser(
		context.Background(),
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, client.Username, user.Username)
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
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
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

	// Create random group
	ciGroup, _, err := client.User.CreateGroup(
		context.Background(),
		&c8y.Group{
			Name: "cigroup" + testingutils.RandomString(7),
		},
	)
	testingutils.Ok(t, err)
	t.Cleanup(func() {
		client.User.DeleteGroup(
			context.Background(),
			ciGroup.GetID(),
		)
	})

	//
	// Get group
	group, resp, err := client.User.GetGroupByName(
		context.Background(),
		ciGroup.Name,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Assert(t, group.GetID() != "", "ID should be greater than 0")

	//
	// Add user to group
	userRef, resp, err := client.User.AddUserToGroup(
		context.Background(),
		currentUser,
		group.GetID(),
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Assert(t, userRef.Self != "", "Self link should not be empty")

	// Get the users in the group
	userReferences, resp, err := client.User.GetUsersByGroup(
		context.Background(),
		group.GetID(),
		&c8y.UserOptions{
			PaginationOptions: *c8y.NewPaginationOptions(100),
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Assert(t, len(userReferences.References) > 0, "Should be at least 1 user")

	//
	// Remove user from group
	resp, err = client.User.RemoveUserFromGroup(
		context.Background(),
		currentUser.Username,
		group.GetID(),
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())
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
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())

	// Create temp group
	name := "group" + testingutils.RandomString(8)
	group, resp, err := client.User.CreateGroup(
		context.Background(),
		&c8y.Group{
			Name: name,
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Equals(t, name, group.Name)

	// Add user to temp group
	_, resp, err = client.User.AddUserToGroup(
		context.Background(),
		currentUser,
		group.GetID(),
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())

	// Get users in group
	userReferences, resp, err := client.User.GetUsersByGroup(
		context.Background(),
		group.GetID(),
		&c8y.UserOptions{
			PaginationOptions: *c8y.NewPaginationOptions(100),
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Assert(t, len(userReferences.References) > 0, "Should be at least one user")
	testingutils.Equals(t, userReferences.References[0].User.Username, currentUser.Username)

	// Update temp group
	updatedName := name + "-UpdatedName"
	updatedGroup, resp, err := client.User.UpdateGroup(
		context.Background(),
		group.GetID(),
		&c8y.Group{
			Name: updatedName,
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, updatedName, updatedGroup.Name)

	// Remove temp group
	resp, err = client.User.DeleteGroup(
		context.Background(),
		group.GetID(),
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())
}

func TestUserService_GetGroups(t *testing.T) {
	client := createTestClient()

	groupCollection, resp, err := client.User.GetGroups(
		context.Background(),
		&c8y.GroupOptions{
			PaginationOptions: *c8y.NewPaginationOptions(100),
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Assert(t, len(groupCollection.Groups) > 0, "Should have at least 1 group reference")
	testingutils.Assert(t, groupCollection.Groups[0].Name != "", "Group reference name should not be empty")
}

func TestUserService_GetGroupsByUser(t *testing.T) {
	client := createTestClient()

	groupCollection, resp, err := client.User.GetGroupsByUser(
		context.Background(),
		client.Username,
		&c8y.GroupOptions{
			PaginationOptions: *c8y.NewPaginationOptions(100),
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Assert(t, len(groupCollection.References) > 0, "Should have at least 1 group reference")
	testingutils.Assert(t, groupCollection.References[0].Group.Name != "", "Group reference name should not be empty")
}

/* ROLES */
func TestUserService_GetRoles(t *testing.T) {
	client := createTestClient()
	roleCollection, resp, err := client.User.GetRoles(
		context.Background(),
		&c8y.RoleOptions{
			PaginationOptions: *c8y.NewPaginationOptions(100),
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Assert(t, len(roleCollection.Roles) > 0, "Should return at least 1 role")
	testingutils.Assert(t, roleCollection.Roles[0].Name != "", "Name should not be empty")
	testingutils.Assert(t, roleCollection.Roles[0].ID != "", "ID should not be empty")
	testingutils.Assert(t, roleCollection.Roles[0].Self != "", "Self should not be empty")
}

func TestUserService_AssignRoleToUser(t *testing.T) {
	client := createTestClient()

	roleCollection, _, err := client.User.GetRoles(
		context.Background(),
		&c8y.RoleOptions{
			PaginationOptions: *c8y.NewPaginationOptions(100),
		},
	)
	testingutils.Ok(t, err)

	// Assign role to user
	roleRef, resp, err := client.User.AssignRoleToUser(
		context.Background(),
		client.Username,
		roleCollection.Roles[0].Self,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Equals(t, roleCollection.Roles[0].Name, roleRef.Role.Name)

	// Get roles by user
	userRoleCollection, resp, err := client.User.GetRolesByUser(
		context.Background(),
		client.Username,
		&c8y.RoleOptions{
			PaginationOptions: *c8y.NewPaginationOptions(100),
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Assert(t, len(userRoleCollection.References) > 0, "Should have at least 1 reference")
	// Check if the role has been assigned to the user
	roleExists := false
	for _, ref := range userRoleCollection.References {
		if roleRef.Role.Name == ref.Role.Name {
			roleExists = true
		}
	}
	testingutils.Equals(t, true, roleExists)

	// Unassign role to user
	resp, err = client.User.UnassignRoleFromUser(
		context.Background(),
		client.Username,
		roleRef.Role.Name,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())
}

func TestUserService_AssignRoleToGroup(t *testing.T) {
	client := createTestClient()

	roleCollection, _, err := client.User.GetRoles(
		context.Background(),
		&c8y.RoleOptions{
			PaginationOptions: *c8y.NewPaginationOptions(100),
		},
	)
	testingutils.Ok(t, err)

	// Get group
	group, resp, err := client.User.GetGroupByName(
		context.Background(),
		"devices",
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())

	testingutils.Assert(t, group != nil, "Group should not be empty")
	if group != nil {
		testingutils.Assert(t, group.Name != "", "Group name should not be empty")
		testingutils.Assert(t, group.GetID() != "", "Group ID should be greater than 0")
	}

	// Remove if role from group if necessary
	// don't worry about the response
	if group != nil && roleCollection != nil && len(roleCollection.Roles) > 0 {
		_, _ = client.User.UnassignRoleFromGroup(
			context.Background(),
			group.GetID(),
			roleCollection.Roles[0].Name,
		)
	}

	// TODO: Use a random role

	// Assign role to user
	roleRef, resp, err := client.User.AssignRoleToGroup(
		context.Background(),
		group.GetID(),
		roleCollection.Roles[0].Self,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Equals(t, roleCollection.Roles[0].Name, roleRef.Role.Name)

	// Get roles by user
	groupRoleCollection, resp, err := client.User.GetRolesByGroup(
		context.Background(),
		group.GetID(),
		&c8y.RoleOptions{
			PaginationOptions: *c8y.NewPaginationOptions(100),
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Assert(t, len(groupRoleCollection.References) > 0, "Should have at least 1 reference")
	// Check if the role has been assigned to the user
	roleExists := false
	for _, ref := range groupRoleCollection.References {
		if roleRef.Role.Name == ref.Role.Name {
			roleExists = true
		}
	}
	testingutils.Equals(t, true, roleExists)

	// Unassign role to user
	resp, err = client.User.UnassignRoleFromGroup(
		context.Background(),
		group.GetID(),
		roleRef.Role.Name,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())
}
