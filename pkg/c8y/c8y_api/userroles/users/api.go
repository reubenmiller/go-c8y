package users

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
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
func (s *Service) AssignRole(ctx context.Context, opt AssignRoleOptions, body any) (*model.UserRoleReference, error) {
	return core.ExecuteResultOnly[model.UserRoleReference](ctx, s.AssignRoleB(opt, body))
}

func (s *Service) AssignRoleB(opt AssignRoleOptions, body any) *core.TryRequest {
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
func (s *Service) UnassignRole(ctx context.Context, opt UnassignRoleOptions) error {
	return core.ExecuteNoResult(ctx, s.UnassignRoleB(opt))
}

func (s *Service) UnassignRoleB(opt UnassignRoleOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamTenantId, opt.TenantID).
		SetPathParam(ParamUserId, opt.UserID).
		SetPathParam(ParamRoleId, opt.RoleID).
		SetURL(ApiUserRole)
	return core.NewTryRequest(s.Client, req)
}
