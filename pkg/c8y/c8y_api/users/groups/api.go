package groups

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiUserGroupUsers = "/user/{tenantID}/groups/{groupId}/users"
var ApiUserGroupReference = "/user/{tenantID}/groups/{groupId}/users/{userId}"

var ParamId = "id"
var ParamTenantId = "tenantID"
var ParamGroupId = "groupId"
var ParamUserId = "userId"

const ResultProperty = "users"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service provides api to manage the current user
type Service struct {
	core.Service
}

type ListUsersOptions struct {
	Tenant  string
	GroupID string
}

// ListUsers in a group
func (s *Service) ListUsers(ctx context.Context, opt ListUsersOptions) (*model.UserReferencesCollection, error) {
	return core.ExecuteResultOnly[model.UserReferencesCollection](ctx, s.ListUsersB(opt))
}

func (s *Service) ListUsersB(opt ListUsersOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, opt.Tenant).
		SetPathParam(ParamGroupId, opt.GroupID).
		SetURL(ApiUserGroupUsers)
	return core.NewTryRequest(s.Client, req)
}

type AssignUserOptions struct {
	Tenant  string
	GroupID string
	User    string
}

// AssignUser assigns a user to a user group
func (s *Service) AssignUser(ctx context.Context, opt AssignUserOptions, user any) (*model.UserReference, error) {
	return core.ExecuteResultOnly[model.UserReference](ctx, s.AssignUserB(opt, user))
}

func (s *Service) AssignUserB(opt AssignUserOptions, user any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(user).
		SetURL(ApiUserGroupUsers)
	return core.NewTryRequest(s.Client, req)
}

type UnassignUserOptions struct {
	Tenant  string `json:"tenant,omitempty"`
	GroupID string `json:"groupId,omitempty"`
	UserID  string `json:"UserId,omitempty"`
}

// UnassignUser unassign a user from a user group
func (s *Service) UnassignUser(ctx context.Context, opt UnassignUserOptions) error {
	return core.ExecuteNoResult(ctx, s.UnassignUserB(opt))
}

func (s *Service) UnassignUserB(opt UnassignUserOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamTenantId, opt.Tenant).
		SetPathParam(ParamGroupId, opt.GroupID).
		SetPathParam(ParamUserId, opt.UserID).
		SetURL(ApiUserGroupUsers)
	return core.NewTryRequest(s.Client, req)
}
