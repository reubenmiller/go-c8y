package api_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/usergroups"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/userroles"
	userrolesgroups "github.com/reubenmiller/go-c8y/pkg/c8y/api/userroles/usergroups"
	userrolesusers "github.com/reubenmiller/go-c8y/pkg/c8y/api/userroles/users"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/users"
	usersgroups "github.com/reubenmiller/go-c8y/pkg/c8y/api/users/groups"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetUsers(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.Users.List(ctx, users.ListOptions{
		Tenant: client.Auth.Tenant,
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	userList, err := op.ToSliceR(result)
	require.NoError(t, err)
	assert.Greater(t, len(userList), 0, "At least 1 user should be present")
}

func Test_GetUser(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.Users.Get(ctx, users.GetOptions{
		ID:     client.Auth.Username,
		Tenant: client.Auth.Tenant,
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)
	assert.Equal(t, client.Auth.Username, result.Data.ID())
}

func Test_GetUserByUsername(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.Users.GetByUsername(ctx, users.GetByUsernameOptions{
		Username: client.Auth.Username,
		Tenant:   client.Auth.Tenant,
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)
	assert.Equal(t, client.Auth.Username, result.Data.ID())
}

func Test_CRUD_User(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "my-ci-user" + testingutils.RandomString(7)
	password := testingutils.RandomPassword(32)

	userInput := model.User{
		Username:  name,
		Email:     name + "@no-reply.org",
		Password:  password,
		FirstName: "User01",
		LastName:  "CI",
	}

	// Create user
	createResult := client.Users.Create(ctx, userInput)

	require.NoError(t, createResult.Err)
	assert.Equal(t, 201, createResult.HTTPStatus)
	assert.Equal(t, userInput.Username, createResult.Data.ID())
	assert.Equal(t, userInput.Username, createResult.Data.UserName())
	assert.Equal(t, userInput.FirstName, createResult.Data.FirstName())
	assert.Equal(t, userInput.LastName, createResult.Data.LastName())

	userID := createResult.Data.ID()

	t.Cleanup(func() {
		client.Users.Delete(ctx, users.DeleteOptions{
			ID:     userID,
			Tenant: client.Auth.Tenant,
		})
	})

	// Update user
	updateResult := client.Users.Update(ctx, users.UpdateOptions{
		ID:     userID,
		Tenant: client.Auth.Tenant,
	}, model.User{
		FirstName: "Alfred",
		LastName:  "Peabody",
		Phone:     "+61 7 1234 5678",
	})

	require.NoError(t, updateResult.Err)
	assert.Equal(t, 200, updateResult.HTTPStatus)
	assert.Equal(t, "Alfred", updateResult.Data.FirstName())
	assert.Equal(t, "Peabody", updateResult.Data.LastName())
	assert.Equal(t, "+61 7 1234 5678", updateResult.Data.Get("phone").String())

	// Delete user
	deleteResult := client.Users.Delete(ctx, users.DeleteOptions{
		ID:     userID,
		Tenant: client.Auth.Tenant,
	})

	require.NoError(t, deleteResult.Err)
	assert.Equal(t, 204, deleteResult.HTTPStatus)
}

func Test_GetCurrentUser(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.Users.CurrentUser.Get(ctx)

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)
	assert.Equal(t, client.Auth.Username, result.Data.UserName())
}

func Test_UpdateCurrentUser(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	randomFirstName := fmt.Sprintf("testUser-%d", time.Now().Unix())
	result := client.Users.CurrentUser.Update(ctx, model.User{
		FirstName: randomFirstName,
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)
	assert.Equal(t, client.Auth.Username, result.Data.UserName())
	assert.Equal(t, randomFirstName, result.Data.FirstName())
}

func Test_GetGroupByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.UserGroups.GetByName(ctx, usergroups.GetByNameOptions{
		GroupName: "admins",
		Tenant:    client.Auth.Tenant,
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)
	assert.Equal(t, "admins", result.Data.Name())

	// Get group by id
	groupID := result.Data.ID()
	getResult := client.UserGroups.Get(ctx, usergroups.Target{
		ID:     groupID,
		Tenant: client.Auth.Tenant,
	})

	require.NoError(t, getResult.Err)
	assert.Equal(t, 200, getResult.HTTPStatus)
	assert.Equal(t, "admins", getResult.Data.Name())
}

func Test_AddUserToGroup(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Get current user
	currentUser := client.Users.GetByUsername(ctx, users.GetByUsernameOptions{
		Username: client.Auth.Username,
		Tenant:   client.Auth.Tenant,
	})
	require.NoError(t, currentUser.Err)

	// Create random group
	groupName := "ci-group" + testingutils.RandomString(7)
	createGroupResult := client.UserGroups.Create(ctx, map[string]any{
		"name": groupName,
	})
	require.NoError(t, createGroupResult.Err)

	groupID := createGroupResult.Data.ID()

	t.Cleanup(func() {
		client.UserGroups.Delete(ctx, usergroups.DeleteOptions{
			Target: usergroups.Target{
				ID:     groupID,
				Tenant: client.Auth.Tenant,
			},
		})
	})

	// Get group
	getGroupResult := client.UserGroups.GetByName(ctx, usergroups.GetByNameOptions{
		GroupName: groupName,
		Tenant:    client.Auth.Tenant,
	})
	require.NoError(t, getGroupResult.Err)
	assert.Equal(t, 200, getGroupResult.HTTPStatus)
	assert.NotEmpty(t, getGroupResult.Data.ID(), "ID should not be empty")

	// Add user to group
	assignResult := client.Users.Groups.AssignUser(ctx, usersgroups.AssignUserOptions{
		GroupID: groupID,
		Tenant:  client.Auth.Tenant,
	}, map[string]any{
		"user": map[string]any{
			"self": currentUser.Data.Self(),
		},
	})

	require.NoError(t, assignResult.Err)
	assert.Equal(t, 201, assignResult.HTTPStatus)
	assert.NotEmpty(t, assignResult.Data.Self(), "Self link should not be empty")

	// Get the users in the group
	listUsersResult := client.Users.Groups.ListUsers(ctx, usersgroups.ListUsersOptions{
		GroupID: groupID,
		Tenant:  client.Auth.Tenant,
	})
	require.NoError(t, listUsersResult.Err)
	assert.Equal(t, 200, listUsersResult.HTTPStatus)

	userList, err := op.ToSliceR(listUsersResult)
	require.NoError(t, err)
	assert.Greater(t, len(userList), 0, "Should be at least 1 user")

	// Remove user from group
	unassignResult := client.Users.Groups.UnassignUser(ctx, usersgroups.UnassignUserOptions{
		UserID:  currentUser.Data.ID(),
		GroupID: groupID,
		Tenant:  client.Auth.Tenant,
	})

	require.NoError(t, unassignResult.Err)
	assert.Equal(t, 204, unassignResult.HTTPStatus)
}

func Test_GetUsersByGroup(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Get current user
	currentUser := client.Users.GetByUsername(ctx, users.GetByUsernameOptions{
		Username: client.Auth.Username,
		Tenant:   client.Auth.Tenant,
	})
	require.NoError(t, currentUser.Err)

	// Create temp group
	name := "group" + testingutils.RandomString(8)
	createResult := client.UserGroups.Create(ctx, map[string]any{
		"name": name,
	})
	require.NoError(t, createResult.Err)
	assert.Equal(t, 201, createResult.HTTPStatus)
	assert.Equal(t, name, createResult.Data.Name())

	groupID := createResult.Data.ID()

	// Add user to temp group
	assignResult := client.Users.Groups.AssignUser(ctx, usersgroups.AssignUserOptions{
		GroupID: groupID,
		Tenant:  client.Auth.Tenant,
	}, map[string]any{
		"user": map[string]any{
			"self": currentUser.Data.Self(),
		},
	})
	require.NoError(t, assignResult.Err)
	assert.Equal(t, 201, assignResult.HTTPStatus)

	// Get users in group
	listUsersResult := client.Users.Groups.ListUsers(ctx, usersgroups.ListUsersOptions{
		GroupID: groupID,
		Tenant:  client.Auth.Tenant,
	})
	require.NoError(t, listUsersResult.Err)
	assert.Equal(t, 200, listUsersResult.HTTPStatus)

	userList, err := op.ToSliceR(listUsersResult)
	require.NoError(t, err)
	assert.Greater(t, len(userList), 0, "Should be at least one user")
	assert.Equal(t, currentUser.Data.UserName(), userList[0].UserName())

	// Update temp group
	updatedName := name + "-UpdatedName"
	updateResult := client.UserGroups.Update(ctx, usergroups.UpdateOptions{
		Target: usergroups.Target{
			ID:     groupID,
			Tenant: client.Auth.Tenant,
		},
	}, map[string]any{
		"name": updatedName,
	})
	require.NoError(t, updateResult.Err)
	assert.Equal(t, 200, updateResult.HTTPStatus)
	assert.Equal(t, updatedName, updateResult.Data.Name())

	// Remove temp group
	deleteResult := client.UserGroups.Delete(ctx, usergroups.DeleteOptions{
		Target: usergroups.Target{
			ID:     groupID,
			Tenant: client.Auth.Tenant,
		},
	})
	require.NoError(t, deleteResult.Err)
	assert.Equal(t, 204, deleteResult.HTTPStatus)
}

func Test_GetGroups(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.UserGroups.List(ctx, usergroups.ListOptions{
		Tenant: client.Auth.Tenant,
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	groupList, err := op.ToSliceR(result)
	require.NoError(t, err)
	assert.Greater(t, len(groupList), 0, "Should have at least 1 group")
	assert.NotEmpty(t, groupList[0].Name(), "Group name should not be empty")
}

func Test_GetGroupsByUser(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.UserGroups.ListByUser(ctx, usergroups.ListByUserOptions{
		UserID: client.Auth.Username,
		Tenant: client.Auth.Tenant,
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	groupList, err := op.ToSliceR(result)
	require.NoError(t, err)
	assert.Greater(t, len(groupList), 0, "Should have at least 1 group")
	assert.NotEmpty(t, groupList[0].Name(), "Group name should not be empty")
}

func Test_GetRoles(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.UserRoles.List(ctx, userroles.ListOptions{})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	roleList, err := op.ToSliceR(result)
	require.NoError(t, err)
	assert.Greater(t, len(roleList), 0, "Should return at least 1 role")
	assert.NotEmpty(t, roleList[0].Name(), "Name should not be empty")
	assert.NotEmpty(t, roleList[0].ID(), "ID should not be empty")
	assert.NotEmpty(t, roleList[0].Self(), "Self should not be empty")
}

func Test_AssignRoleToUser(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	rolesResult := client.UserRoles.List(ctx, userroles.ListOptions{})
	require.NoError(t, rolesResult.Err)

	roles, err := op.ToSliceR(rolesResult)
	require.NoError(t, err)
	require.Greater(t, len(roles), 0, "Should have at least 1 role")

	// Assign role to user
	assignResult := client.UserRoles.Users.AssignRole(ctx, userrolesusers.AssignRoleOptions{
		UserID:   client.Auth.Username,
		TenantID: client.Auth.Tenant,
	}, map[string]any{
		"role": map[string]any{
			"self": roles[0].Self(),
		},
	})
	require.NoError(t, assignResult.Err)
	assert.Equal(t, 201, assignResult.HTTPStatus)
	assert.Equal(t, roles[0].Name(), assignResult.Data.Name())

	// Get roles by user using ListGroupsWithUser to verify
	groupsResult := client.Users.ListGroupsWithUser(ctx, users.ListGroupsOptions{
		UserID: client.Auth.Username,
		Tenant: client.Auth.Tenant,
	})
	require.NoError(t, groupsResult.Err)
	assert.Equal(t, 200, groupsResult.HTTPStatus)
	assert.GreaterOrEqual(t, groupsResult.Data.Length(), 1)
	for item := range op.Iter(groupsResult) {
		val := item.ID()
		require.NotEmpty(t, val)
	}

	// Unassign role from user
	unassignResult := client.UserRoles.Users.UnassignRole(ctx, userrolesusers.UnassignRoleOptions{
		UserID:   client.Auth.Username,
		RoleID:   assignResult.Data.ID(),
		TenantID: client.Auth.Tenant,
	})
	require.NoError(t, unassignResult.Err)
	assert.Equal(t, 204, unassignResult.HTTPStatus)
}

func Test_AssignRoleToGroup(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	rolesResult := client.UserRoles.List(ctx, userroles.ListOptions{})
	require.NoError(t, rolesResult.Err)

	roles, err := op.ToSliceR(rolesResult)
	require.NoError(t, err)
	require.Greater(t, len(roles), 0, "Should have at least 1 role")

	// Get group
	groupResult := client.UserGroups.GetByName(ctx, usergroups.GetByNameOptions{
		GroupName: "devices",
		Tenant:    client.Auth.Tenant,
	})
	require.NoError(t, groupResult.Err)
	assert.Equal(t, 200, groupResult.HTTPStatus)
	assert.NotEmpty(t, groupResult.Data.Name(), "Group name should not be empty")
	assert.NotEmpty(t, groupResult.Data.ID(), "Group ID should not be empty")

	groupID := groupResult.Data.ID()

	// Remove role from group if necessary (don't worry about the response)
	client.UserRoles.Groups.UnassignRole(ctx, userrolesgroups.UnassignRoleOptions{
		GroupID:  groupID,
		RoleID:   roles[0].ID(),
		TenantID: client.Auth.Tenant,
	})

	// Assign role to group
	assignResult := client.UserRoles.Groups.AssignRole(ctx, userrolesgroups.AssignRoleOptions{
		GroupID:  groupID,
		TenantID: client.Auth.Tenant,
	}, map[string]any{
		"role": map[string]any{
			"self": roles[0].Self(),
		},
	})
	require.NoError(t, assignResult.Err)
	assert.Equal(t, 201, assignResult.HTTPStatus)
	assert.Equal(t, roles[0].Name(), assignResult.Data.Name())

	// Get roles by group
	groupRolesResult := client.UserRoles.Groups.ListRoles(ctx, userrolesgroups.ListRolesOptions{
		UserGroupID: groupID,
		Tenant:      client.Auth.Tenant,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 2000,
		},
	})
	require.NoError(t, groupRolesResult.Err)
	assert.Equal(t, 200, groupRolesResult.HTTPStatus)

	groupRoles, err := op.ToSliceR(groupRolesResult)
	require.NoError(t, err)
	assert.Greater(t, len(groupRoles), 0, "Should have at least 1 role")

	// Check if the role has been assigned to the group
	roleExists := false
	for _, role := range groupRoles {
		if assignResult.Data.Name() == role.Name() {
			roleExists = true
			break
		}
	}
	assert.True(t, roleExists, "Role should be assigned to group")

	// Unassign role from group
	unassignResult := client.UserRoles.Groups.UnassignRole(ctx, userrolesgroups.UnassignRoleOptions{
		GroupID:  groupID,
		RoleID:   assignResult.Data.ID(),
		TenantID: client.Auth.Tenant,
	})
	require.NoError(t, unassignResult.Err)
	assert.Equal(t, 204, unassignResult.HTTPStatus)
}
