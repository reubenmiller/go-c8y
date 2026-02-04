package groups

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
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
func (s *Service) ListUsers(ctx context.Context, opt ListUsersOptions) op.Result[jsonmodels.User] {
	return core.ExecuteReturnCollection(ctx, s.listUsersB(opt), "references", "", func(b []byte) jsonmodels.User {
		// Extract user from reference wrapper
		doc := jsondoc.New(b)
		return jsonmodels.NewUser([]byte(doc.Get("user").Raw))
	})
}

func (s *Service) listUsersB(opt ListUsersOptions) *core.TryRequest {
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
func (s *Service) AssignUser(ctx context.Context, opt AssignUserOptions, user any) op.Result[jsonmodels.User] {
	return core.ExecuteReturnResult(ctx, s.assignUserB(opt, user), func(b []byte) jsonmodels.User {
		// Extract user from reference wrapper
		doc := jsondoc.New(b)
		return jsonmodels.NewUser([]byte(doc.Get("user").Raw))
	})
}

func (s *Service) assignUserB(opt AssignUserOptions, user any) *core.TryRequest {
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
func (s *Service) UnassignUser(ctx context.Context, opt UnassignUserOptions) op.Result[core.NoContent] {
	return core.ExecuteNoResult(ctx, s.unassignUserB(opt))
}

func (s *Service) unassignUserB(opt UnassignUserOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamTenantId, opt.Tenant).
		SetPathParam(ParamGroupId, opt.GroupID).
		SetPathParam(ParamUserId, opt.UserID).
		SetURL(ApiUserGroupUsers)
	return core.NewTryRequest(s.Client, req)
}
