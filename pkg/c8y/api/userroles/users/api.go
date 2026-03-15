package users

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiUserRoles = "/user/{tenantId}/users/{userId}/roles"
var ApiUserRole = "/user/{tenantId}/users/{userId}/roles/{roleId}"

var ParamUserId = "userId"
var ParamRoleId = "roleId"

// ResultProperty is the JSON path used to extract roles from the role reference collection response.
// The response format is: { references: [{ role: {...} }] }
const ResultProperty = "references.#.role"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service provides api to manage user roles
type Service struct {
	core.Service
}

// ListOptions to list roles assigned to a specific user
type ListOptions struct {
	// TenantID is the tenant the user belongs to. Defaults to the current tenant.
	TenantID string `url:"-"`
	// UserID is the ID of the user whose roles are listed.
	UserID string `url:"-"`
	pagination.PaginationOptions
}

// RoleIterator provides iteration over roles assigned to a user
type RoleIterator = pagination.Iterator[jsonmodels.Role]

// List retrieves all roles assigned to a specific user (by a given user ID) in a specific tenant (by a given tenant ID).
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Role] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewRole)
}

// ListAll returns an iterator for all roles assigned to a user, automatically paginating.
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *RoleIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Role] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewRole,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(ParamUserId, opt.UserID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiUserRoles)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type AssignRoleOptions struct {
	TenantID string `url:"-"`
	UserID   string `url:"-"`
}

// AssignRole assigns a role to a user
func (s *Service) AssignRole(ctx context.Context, opt AssignRoleOptions, body any) op.Result[jsonmodels.Role] {
	return core.Execute(ctx, s.assignRoleB(opt, body), func(b []byte) jsonmodels.Role {
		// Extract role from reference wrapper
		doc := jsondoc.New(b)
		return jsonmodels.NewRole([]byte(doc.Get("role").Raw))
	}).IgnoreConflict()
}

func (s *Service) assignRoleB(opt AssignRoleOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(ParamUserId, opt.UserID).
		SetBody(body).
		SetURL(ApiUserRoles)
	return core.NewTryRequest(s.Client, req)
}

type UnassignRoleOptions struct {
	TenantID string `url:"-"`
	UserID   string `url:"-"`
	RoleID   string `url:"-"`
}

// Unassign a role from a user
func (s *Service) UnassignRole(ctx context.Context, opt UnassignRoleOptions) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.unassignRoleB(opt)).IgnoreNotFound()
}

func (s *Service) unassignRoleB(opt UnassignRoleOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(ParamUserId, opt.UserID).
		SetPathParam(ParamRoleId, opt.RoleID).
		SetURL(ApiUserRole)
	return core.NewTryRequest(s.Client, req)
}
