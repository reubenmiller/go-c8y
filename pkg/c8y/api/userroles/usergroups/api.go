package usergroups

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiGroupRoles = "/user/{tenantId}/groups/{groupId}/roles"
var ApiGroupRole = "/user/{tenantId}/groups/{groupId}/roles/{roleId}"

var ParamGroupId = "groupId"
var ParamRoleId = "roleId"

// ResultProperty is the JSON path used to extract the role references from the
// role reference collection response. The response format is:
// { references: [{ self: ..., role: {...} }] }
const ResultProperty = "references"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service provides api to manage the roles assigned to a user group.
type Service struct {
	core.Service
}

// ListOptions to list the role references assigned to a specific user group.
type ListOptions struct {
	// TenantID is the tenant the group belongs to. Defaults to the current tenant.
	TenantID string `url:"-"`
	// GroupID is the ID of the group whose role references are listed.
	GroupID string `url:"-"`
	pagination.PaginationOptions
}

// RoleReferenceIterator provides iteration over the role references assigned to a group.
type RoleReferenceIterator = pagination.Iterator[jsonmodels.RoleReference]

// List retrieves the role references assigned to a specific user group (by a given group ID)
// in a specific tenant (by a given tenant ID).
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.RoleReference] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewRoleReference)
}

// ListAll returns an iterator over all role references assigned to a group, automatically paginating.
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *RoleReferenceIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.RoleReference] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewRoleReference,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(ParamGroupId, opt.GroupID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiGroupRoles)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type AssignRoleOptions struct {
	TenantID string `url:"-"`
	GroupID  string `url:"-"`
}

// AssignRole assigns a role to a user group. The body is the role reference document
// ({ "role": { "self": "<role self link>" } }); the created reference is returned.
// An already-assigned role (409 Conflict) is reported as a duplicate, not an error.
func (s *Service) AssignRole(ctx context.Context, opt AssignRoleOptions, body any) op.Result[jsonmodels.RoleReference] {
	return core.Execute(ctx, s.assignRoleB(opt, body), jsonmodels.NewRoleReference).IgnoreConflict()
}

func (s *Service) assignRoleB(opt AssignRoleOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(ParamGroupId, opt.GroupID).
		SetBody(body).
		SetURL(ApiGroupRoles)
	return core.NewTryRequest(s.Client, req)
}

type UnassignRoleOptions struct {
	TenantID string `url:"-"`
	GroupID  string `url:"-"`
	RoleID   string `url:"-"`
}

// UnassignRole removes a role from a user group. A missing role reference (404) is
// reported as skipped, not an error.
func (s *Service) UnassignRole(ctx context.Context, opt UnassignRoleOptions) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.unassignRoleB(opt)).IgnoreNotFound()
}

func (s *Service) unassignRoleB(opt UnassignRoleOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(ParamRoleId, opt.RoleID).
		SetPathParam(ParamGroupId, opt.GroupID).
		SetURL(ApiGroupRole)
	return core.NewTryRequest(s.Client, req)
}
