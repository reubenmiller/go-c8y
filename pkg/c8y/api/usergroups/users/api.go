// Package users manages the users that belong to a user group: the member
// collection hanging off a group (/user/{tenant}/groups/{group}/users). It is
// the group-side counterpart of the role reference services (UserRoles.Groups /
// UserRoles.Users) and is wired on the client as UserGroups.Users.
package users

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiGroupUsers = "/user/{tenantId}/groups/{groupId}/users"
var ApiGroupUser = "/user/{tenantId}/groups/{groupId}/users/{userId}"

var ParamGroupId = "groupId"
var ParamUserId = "userId"

// ResultProperty is the JSON path used to extract the member users from a user
// reference collection response. The response format is:
//
//	{ references: [{ self: ..., user: {...} }] }
//
// Each item is plucked down to its embedded user, matching the v1
// listGroupMembership surface (a collection of users, not of references).
const ResultProperty = "references.#.user"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service provides api to manage the users that belong to a user group.
type Service struct {
	core.Service
}

// ListOptions to list the users belonging to a specific user group.
type ListOptions struct {
	// TenantID is the tenant the group belongs to. Defaults to the current tenant.
	TenantID string `url:"-"`
	// GroupID is the ID of the group whose member users are listed.
	GroupID string `url:"-"`
	pagination.PaginationOptions
}

// UserIterator provides iteration over the users belonging to a group.
type UserIterator = pagination.Iterator[jsonmodels.User]

// List retrieves the users belonging to a specific user group (by a given group
// ID) in a specific tenant (by a given tenant ID). Each item is the embedded
// user; the reference wrapper is unwrapped.
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.User] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewUser)
}

// ListAll returns an iterator over all users belonging to a group, automatically paginating.
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *UserIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.User] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewUser,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(ParamGroupId, opt.GroupID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiGroupUsers)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type AssignUserOptions struct {
	TenantID string `url:"-"`
	GroupID  string `url:"-"`
}

// AssignUser adds a user to a user group. The body is the user reference document
// ({ "user": { "self": "<user self link>" } }); the created reference is returned.
// An already-assigned user (409 Conflict) is reported as a duplicate, not an error.
func (s *Service) AssignUser(ctx context.Context, opt AssignUserOptions, body any) op.Result[jsonmodels.UserReference] {
	return core.Execute(ctx, s.assignUserB(opt, body), jsonmodels.NewUserReference).IgnoreConflict()
}

func (s *Service) assignUserB(opt AssignUserOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeUserReference).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(ParamGroupId, opt.GroupID).
		SetBody(body).
		SetURL(ApiGroupUsers)
	return core.NewTryRequest(s.Client, req)
}

type UnassignUserOptions struct {
	TenantID string `url:"-"`
	GroupID  string `url:"-"`
	UserID   string `url:"-"`
}

// UnassignUser removes a user from a user group. A missing membership (404) is
// reported as skipped, not an error.
func (s *Service) UnassignUser(ctx context.Context, opt UnassignUserOptions) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.unassignUserB(opt)).IgnoreNotFound()
}

func (s *Service) unassignUserB(opt UnassignUserOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(ParamGroupId, opt.GroupID).
		SetPathParam(ParamUserId, opt.UserID).
		SetURL(ApiGroupUser)
	return core.NewTryRequest(s.Client, req)
}
