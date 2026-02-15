package users

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiUserRoles = "/user/{tenantID}/users/{userId}/roles"
var ApiUserRole = "/user/{tenantID}/users/{userId}/roles/{roleId}"

var ParamUserId = "userId"
var ParamRoleId = "roleId"
var ParamTenantId = "tenantID"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service provides api to manage user roles
type Service struct {
	core.Service
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
	})
}

func (s *Service) assignRoleB(opt AssignRoleOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, opt.TenantID).
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
	return core.ExecuteNoContent(ctx, s.unassignRoleB(opt))
}

func (s *Service) unassignRoleB(opt UnassignRoleOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamTenantId, opt.TenantID).
		SetPathParam(ParamUserId, opt.UserID).
		SetPathParam(ParamRoleId, opt.RoleID).
		SetURL(ApiUserRole)
	return core.NewTryRequest(s.Client, req)
}
