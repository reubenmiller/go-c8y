package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/usergroups"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/userroles"
	rolegroups "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/userroles/usergroups"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/users"
	groupusers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/users/groups"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// SyncUserGroups ensures user groups (called "global roles" in the Cumulocity
// UI) exist with the desired description and roles. Role assignments are
// additive: roles already on the group but absent from the manifest are left
// alone, so built-in groups (admins, business, ...) can be extended safely.
func (s *Syncer) SyncUserGroups(ctx context.Context, specs []UserGroupSpec) error {
	// Role self links are shared by all groups of the target tenant
	roleSelfLinks := map[string]string{}

	for _, spec := range specs {
		if s.DryRun {
			detail := "ensure group"
			if len(spec.Roles) > 0 {
				detail += " with roles: " + strings.Join(spec.Roles, ", ")
			}
			s.record(SectionUserGroups, spec.Name, ActionPlanned, detail, nil)
			continue
		}

		group := s.Client.UserGroups.GetByName(ctx, usergroups.GetByNameOptions{GroupName: spec.Name})

		action := ActionUnchanged
		var details []string
		var groupID string

		if group.Err != nil {
			// Not found (or not readable): create it
			body := map[string]any{"name": spec.Name}
			if spec.Description != "" {
				body["description"] = spec.Description
			}
			created := s.Client.UserGroups.Create(ctx, body)
			if created.Err != nil {
				s.record(SectionUserGroups, spec.Name, ActionFailed, "create group", created.Err)
				continue
			}
			groupID = created.Data.ID()
			action = ActionCreated
		} else {
			groupID = group.Data.ID()
			if spec.Description != "" && group.Data.Get("description").String() != spec.Description {
				result := s.Client.UserGroups.Update(ctx, usergroups.UpdateOptions{ID: usergroups.ByID(groupID)}, map[string]any{
					"name":        spec.Name,
					"description": spec.Description,
				})
				if result.Err != nil {
					s.record(SectionUserGroups, spec.Name, ActionFailed, "update group", result.Err)
					continue
				}
				action = ActionUpdated
				details = append(details, "description updated")
			}
		}

		assigned, err := s.ensureGroupRoles(ctx, groupID, action == ActionCreated, spec.Roles, roleSelfLinks)
		if err != nil {
			s.record(SectionUserGroups, spec.Name, ActionFailed, "assign roles", err)
			continue
		}
		if len(assigned) > 0 {
			if action == ActionUnchanged {
				action = ActionUpdated
			}
			details = append(details, "assigned roles: "+strings.Join(assigned, ", "))
		}
		s.record(SectionUserGroups, spec.Name, action, strings.Join(details, "; "), nil)
	}
	return nil
}

// ensureGroupRoles assigns the missing roles to a group and returns the names
// of the roles actually assigned. For a freshly created group the existing
// role lookup is skipped (it has none yet).
func (s *Syncer) ensureGroupRoles(ctx context.Context, groupID string, created bool, roles []string, roleSelfLinks map[string]string) ([]string, error) {
	if len(roles) == 0 {
		return nil, nil
	}

	existing := map[string]bool{}
	if !created {
		result := s.Client.UserRoles.Groups.ListRoles(ctx, rolegroups.ListRolesOptions{
			UserGroupID:       groupID,
			PaginationOptions: pagination.PaginationOptions{PageSize: 2000},
		})
		for item, err := range op.Iter2(result) {
			if err != nil {
				return nil, fmt.Errorf("list roles: %w", err)
			}
			existing[item.Name()] = true
		}
	}

	var assigned []string
	for _, roleName := range roles {
		if existing[roleName] {
			continue
		}
		selfLink, ok := roleSelfLinks[roleName]
		if !ok {
			role := s.Client.UserRoles.Get(ctx, userroles.GetOption{Name: roleName})
			if role.Err != nil {
				return nil, fmt.Errorf("role %q: %w", roleName, role.Err)
			}
			selfLink = role.Data.Self()
			roleSelfLinks[roleName] = selfLink
		}
		result := s.Client.UserRoles.Groups.AssignRole(ctx, rolegroups.AssignRoleOptions{GroupID: groupID}, map[string]any{
			"role": map[string]any{"self": selfLink},
		})
		if result.Err != nil {
			return nil, fmt.Errorf("role %q: %w", roleName, result.Err)
		}
		// A conflict means the role was already assigned
		if result.Status != op.StatusDuplicate {
			assigned = append(assigned, roleName)
		}
	}
	return assigned, nil
}

// SyncUsers ensures users exist with the desired profile fields and group
// memberships. The password is only used when creating a user (existing
// passwords are never overwritten); group memberships are additive: groups
// the user belongs to but which are absent from the manifest are kept.
func (s *Syncer) SyncUsers(ctx context.Context, specs []UserSpec) error {
	// Group IDs are resolved once and shared by all users of the target tenant
	groupIDs := map[string]string{}

	for _, spec := range specs {
		if s.DryRun {
			detail := "ensure user"
			if len(spec.Groups) > 0 {
				detail += " in groups: " + strings.Join(spec.Groups, ", ")
			}
			s.record(SectionUsers, spec.Username, ActionPlanned, detail, nil)
			continue
		}

		existing := s.Client.Users.GetByUsername(ctx, users.GetByUsernameOptions{Username: spec.Username})

		action := ActionUnchanged
		var details []string
		var userID, userSelf string

		if existing.Err != nil {
			// Not found: create it
			created := s.Client.Users.Create(ctx, userCreateBody(spec))
			if created.Err != nil {
				s.record(SectionUsers, spec.Username, ActionFailed, "create user", created.Err)
				continue
			}
			userID = created.Data.ID()
			userSelf = created.Data.Self()
			action = ActionCreated
		} else {
			userID = existing.Data.ID()
			userSelf = existing.Data.Self()
			if changes := userDiff(spec, existing.Data); len(changes) > 0 {
				result := s.Client.Users.Update(ctx, users.UpdateOptions{ID: users.ByID(userID)}, changes)
				if result.Err != nil {
					s.record(SectionUsers, spec.Username, ActionFailed, "update user", result.Err)
					continue
				}
				action = ActionUpdated
				details = append(details, "updated "+strings.Join(sortedKeys(changes), ", "))
			}
		}

		assigned, err := s.ensureUserGroups(ctx, userID, userSelf, action == ActionCreated, spec.Groups, groupIDs)
		if err != nil {
			s.record(SectionUsers, spec.Username, ActionFailed, "assign groups", err)
			continue
		}
		if len(assigned) > 0 {
			if action == ActionUnchanged {
				action = ActionUpdated
			}
			details = append(details, "assigned to groups: "+strings.Join(assigned, ", "))
		}
		s.record(SectionUsers, spec.Username, action, strings.Join(details, "; "), nil)
	}
	return nil
}

// ensureUserGroups assigns the user to the missing groups and returns the
// names of the groups the user was actually assigned to. For a freshly
// created user the membership lookup is skipped (it has none yet).
func (s *Syncer) ensureUserGroups(ctx context.Context, userID, userSelf string, created bool, groups []string, groupIDs map[string]string) ([]string, error) {
	if len(groups) == 0 {
		return nil, nil
	}

	member := map[string]bool{}
	if !created {
		result := s.Client.Users.ListGroupsWithUser(ctx, users.ListGroupsOptions{
			UserID:            userID,
			PaginationOptions: pagination.PaginationOptions{PageSize: 2000},
		})
		for item, err := range op.Iter2(result) {
			if err != nil {
				return nil, fmt.Errorf("list group memberships: %w", err)
			}
			member[item.Name()] = true
		}
	}

	var assigned []string
	for _, groupName := range groups {
		if member[groupName] {
			continue
		}
		groupID, ok := groupIDs[groupName]
		if !ok {
			group := s.Client.UserGroups.GetByName(ctx, usergroups.GetByNameOptions{GroupName: groupName})
			if group.Err != nil {
				return nil, fmt.Errorf("group %q: %w", groupName, group.Err)
			}
			groupID = group.Data.ID()
			groupIDs[groupName] = groupID
		}
		result := s.Client.Users.Groups.AssignUser(ctx, groupusers.AssignUserOptions{GroupID: groupID}, map[string]any{
			"user": map[string]any{"self": userSelf},
		})
		if result.Err != nil {
			return nil, fmt.Errorf("group %q: %w", groupName, result.Err)
		}
		// A conflict means the user was already a member
		if result.Status != op.StatusDuplicate {
			assigned = append(assigned, groupName)
		}
	}
	return assigned, nil
}

// userCreateBody builds the create request body for a user
func userCreateBody(spec UserSpec) map[string]any {
	body := map[string]any{
		"userName": spec.Username,
		"enabled":  spec.IsEnabled(),
	}
	if spec.Email != "" {
		body["email"] = spec.Email
	}
	if spec.FirstName != "" {
		body["firstName"] = spec.FirstName
	}
	if spec.LastName != "" {
		body["lastName"] = spec.LastName
	}
	if spec.Phone != "" {
		body["phone"] = spec.Phone
	}
	if spec.Password != "" {
		body["password"] = spec.Password
	}
	if spec.SendPasswordResetEmail {
		body["sendPasswordResetEmail"] = true
	}
	return body
}

// userDiff returns the fields whose existing value differs from the manifest.
// Only fields set in the manifest are managed (empty fields are left alone),
// except enabled which defaults to true. The password is deliberately never
// part of an update.
func userDiff(spec UserSpec, existing jsonmodels.User) map[string]any {
	changes := map[string]any{}
	if spec.Email != "" && existing.Email() != spec.Email {
		changes["email"] = spec.Email
	}
	if spec.FirstName != "" && existing.FirstName() != spec.FirstName {
		changes["firstName"] = spec.FirstName
	}
	if spec.LastName != "" && existing.LastName() != spec.LastName {
		changes["lastName"] = spec.LastName
	}
	if spec.Phone != "" && existing.Get("phone").String() != spec.Phone {
		changes["phone"] = spec.Phone
	}
	if existing.Enabled() != spec.IsEnabled() {
		changes["enabled"] = spec.IsEnabled()
	}
	return changes
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
