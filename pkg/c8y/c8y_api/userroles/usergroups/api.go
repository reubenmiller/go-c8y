package usergroups

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/users/currentuser"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/users/groups"
	"resty.dev/v3"
)

var ApiGroupRoles = "/user/{tenantID}/groups/{groupId}/roles"
var ApiGroupRole = "/user/{tenantID}/groups/{groupId}/roles/{roleId}"

var ParamGroupId = "groupId"
var ParamRoleId = "roleId"
var ParamTenantId = "tenantID"

const ResultProperty = "references"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service provides api to manage user roles
type Service struct {
	core.Service

	CurrentUser *currentuser.Service
	Groups      *groups.Service
}

// ListRolesOptions to filter the user groups which contain a specific role
type ListRolesOptions struct {
	// Defaults to the current tenant
	Tenant string `url:"-"`

	UserGroupID string `url:"-"`

	pagination.PaginationOptions
}

// Retrieve all roles assigned to a specific user group (by a given user group ID) in a specific tenant (by a given tenant ID)
func (s *Service) ListRoles(ctx context.Context, opt ListRolesOptions) op.Result[jsonmodels.Role] {
	return core.ExecuteReturnCollection(ctx, s.ListRolesB(opt), ResultProperty, types.ResponseFieldStatistics, func(b []byte) jsonmodels.Role {
		// Extract role from reference wrapper
		doc := jsondoc.New(b)
		return jsonmodels.NewRole([]byte(doc.Get("role").Raw))
	})
}

func (s *Service) ListRolesB(opt ListRolesOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, opt.Tenant).
		SetPathParam(ParamGroupId, opt.UserGroupID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiGroupRoles)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type AssignRoleOptions struct {
	TenantID string `url:"-"`
	GroupID  string `url:"-"`
}

// AssignRole assigns a role to a user group
func (s *Service) AssignRole(ctx context.Context, opt AssignRoleOptions, body any) op.Result[jsonmodels.Role] {
	return core.ExecuteReturnResult(ctx, s.AssignRoleB(opt, body), func(b []byte) jsonmodels.Role {
		// Extract role from reference wrapper
		doc := jsondoc.New(b)
		return jsonmodels.NewRole([]byte(doc.Get("role").Raw))
	})
}

func (s *Service) AssignRoleB(opt AssignRoleOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, opt.TenantID).
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

// Unassign a role from a user group
func (s *Service) UnassignRole(ctx context.Context, opt UnassignRoleOptions) op.Result[core.NoContent] {
	return core.ExecuteNoResult(ctx, s.UnassignRoleB(opt))
}

func (s *Service) UnassignRoleB(opt UnassignRoleOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamTenantId, opt.TenantID).
		SetPathParam(ParamRoleId, opt.RoleID).
		SetPathParam(ParamGroupId, opt.GroupID).
		SetURL(ApiGroupRole)
	return core.NewTryRequest(s.Client, req)
}
