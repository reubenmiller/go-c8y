package main

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadManifestUsersAndGroups(t *testing.T) {
	path := writeManifest(t, `
userGroups:
  - name: operators
    description: Operations team
    roles: [ROLE_INVENTORY_READ, ROLE_ALARM_READ]

users:
  - userName: jdoe@example.com
    email: jdoe@example.com
    firstName: Jane
    lastName: Doe
    sendPasswordResetEmail: true
    groups: [operators, admins]
  - userName: disabled-account
    enabled: false
`)
	manifest, err := LoadManifest(path)
	require.NoError(t, err)

	require.Len(t, manifest.UserGroups, 1)
	assert.Equal(t, "operators", manifest.UserGroups[0].Name)
	assert.Equal(t, []string{"ROLE_INVENTORY_READ", "ROLE_ALARM_READ"}, manifest.UserGroups[0].Roles)

	require.Len(t, manifest.Users, 2)
	assert.Equal(t, "jdoe@example.com", manifest.Users[0].Username)
	assert.Equal(t, []string{"operators", "admins"}, manifest.Users[0].Groups)
	assert.True(t, manifest.Users[0].IsEnabled())
	assert.True(t, manifest.Users[0].SendPasswordResetEmail)
	assert.False(t, manifest.Users[1].IsEnabled())
}

func TestSyncUserGroupsDryRun(t *testing.T) {
	syncer := &Syncer{DryRun: true}

	err := syncer.SyncUserGroups(context.Background(), []UserGroupSpec{
		{Name: "operators", Roles: []string{"ROLE_INVENTORY_READ", "ROLE_ALARM_READ"}},
		{Name: "empty-group"},
	})
	require.NoError(t, err)

	require.Len(t, syncer.Results, 2)
	assert.Equal(t, ActionPlanned, syncer.Results[0].Action)
	assert.Equal(t, SectionUserGroups, syncer.Results[0].Section)
	assert.Equal(t, "operators", syncer.Results[0].Item)
	assert.Equal(t, "ensure group with roles: ROLE_INVENTORY_READ, ROLE_ALARM_READ", syncer.Results[0].Detail)
	assert.Equal(t, "ensure group", syncer.Results[1].Detail)
}

func TestSyncUsersDryRun(t *testing.T) {
	syncer := &Syncer{DryRun: true}

	err := syncer.SyncUsers(context.Background(), []UserSpec{
		{Username: "jdoe@example.com", Groups: []string{"operators", "admins"}},
		{Username: "svc-account"},
	})
	require.NoError(t, err)

	require.Len(t, syncer.Results, 2)
	assert.Equal(t, ActionPlanned, syncer.Results[0].Action)
	assert.Equal(t, SectionUsers, syncer.Results[0].Section)
	assert.Equal(t, "jdoe@example.com", syncer.Results[0].Item)
	assert.Equal(t, "ensure user in groups: operators, admins", syncer.Results[0].Detail)
	assert.Equal(t, "ensure user", syncer.Results[1].Detail)
}

func TestUserCreateBody(t *testing.T) {
	t.Run("full spec", func(t *testing.T) {
		disabled := false
		body := userCreateBody(UserSpec{
			Username:               "jdoe@example.com",
			Email:                  "jdoe@example.com",
			FirstName:              "Jane",
			LastName:               "Doe",
			Phone:                  "+1234567890",
			Enabled:                &disabled,
			Password:               "s3cret-Passw0rd",
			SendPasswordResetEmail: true,
		})
		assert.Equal(t, map[string]any{
			"userName":               "jdoe@example.com",
			"email":                  "jdoe@example.com",
			"firstName":              "Jane",
			"lastName":               "Doe",
			"phone":                  "+1234567890",
			"enabled":                false,
			"password":               "s3cret-Passw0rd",
			"sendPasswordResetEmail": true,
		}, body)
	})

	t.Run("minimal spec defaults to enabled", func(t *testing.T) {
		body := userCreateBody(UserSpec{Username: "svc-account"})
		assert.Equal(t, map[string]any{
			"userName": "svc-account",
			"enabled":  true,
		}, body)
	})
}

func TestUserDiff(t *testing.T) {
	existing := jsonmodels.NewUser([]byte(`{
		"id": "jdoe@example.com",
		"userName": "jdoe@example.com",
		"email": "jdoe@example.com",
		"firstName": "Jane",
		"lastName": "Doe",
		"enabled": true
	}`))

	t.Run("unchanged", func(t *testing.T) {
		changes := userDiff(UserSpec{
			Username:  "jdoe@example.com",
			Email:     "jdoe@example.com",
			FirstName: "Jane",
		}, existing)
		assert.Empty(t, changes)
	})

	t.Run("fields not in the manifest are left alone", func(t *testing.T) {
		changes := userDiff(UserSpec{Username: "jdoe@example.com"}, existing)
		assert.Empty(t, changes)
	})

	t.Run("changed fields are updated", func(t *testing.T) {
		disabled := false
		changes := userDiff(UserSpec{
			Username: "jdoe@example.com",
			Email:    "new@example.com",
			LastName: "Smith",
			Phone:    "+1234567890",
			Enabled:  &disabled,
		}, existing)
		assert.Equal(t, map[string]any{
			"email":    "new@example.com",
			"lastName": "Smith",
			"phone":    "+1234567890",
			"enabled":  false,
		}, changes)
	})

	t.Run("password is never part of an update", func(t *testing.T) {
		changes := userDiff(UserSpec{
			Username: "jdoe@example.com",
			Password: "new-Passw0rd!",
		}, existing)
		assert.Empty(t, changes)
	})
}
