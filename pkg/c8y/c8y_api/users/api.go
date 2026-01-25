package users

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/users/currentuser"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/users/groups"
	"resty.dev/v3"
)

var (
	ApiUsers              = "/user/{tenantID}/users"
	ApiUser               = "/user/{tenantID}/users/{id}"
	ApiUserGroupsWithUser = "/user/{tenantID}/users/{id}/groups"
	ApiUserByName         = "/user/{tenantID}/userByName/{username}"
)

var ParamId = "id"
var ParamUsername = "username"
var ParamTenantId = "tenantID"

const ResultProperty = "users"

func NewService(s *core.Service) *Service {
	return &Service{
		Service:     *s,
		CurrentUser: currentuser.NewService(s),
		Groups:      groups.NewService(s),
	}
}

// Service provides api to manage users
type Service struct {
	core.Service

	CurrentUser *currentuser.Service
	Groups      *groups.Service
}

// ListOptions to filter the users by
type ListOptions struct {
	// Defaults to the current tenant
	Tenant string `url:"-"`

	Username string `url:"username,omitempty"`

	Groups []string `url:"groups,omitempty"`

	// Exact username
	Owner string `url:"owner,omitempty"`

	// OnlyDevices If set to "true", result will contain only users created during bootstrap process (starting with "device_"). If flag is absent (or false) the result will not contain "device_" users.
	OnlyDevices bool `url:"onlyDevices,omitempty"`

	// WithSubusersCount if set to "true", then each of returned users will contain additional field "subusersCount" - number of direct subusers (users with corresponding "owner").
	WithSubusersCount bool `url:"withSubusersCount,omitempty"`

	pagination.PaginationOptions
}

// Retrieve all users in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) (*model.UserCollection, error) {
	return core.ExecuteResultOnly[model.UserCollection](ctx, s.ListB(opt))
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, opt.Tenant).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiUsers)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type Target struct {
	ID     string
	Tenant string
}

// Get a user
func (s *Service) Get(ctx context.Context, target Target) (*model.User, error) {
	return core.ExecuteResultOnly[model.User](ctx, s.GetB(target))
}

func (s *Service) GetB(target Target) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, target.Tenant).
		SetPathParam(ParamId, target.ID).
		SetURL(ApiUser)
	return core.NewTryRequest(s.Client, req)
}

type GetByUsernameOptions struct {
	Username string
	Tenant   string
}

// Get a user by username
func (s *Service) GetByUsername(ctx context.Context, opt GetByUsernameOptions) (*model.User, error) {
	return core.ExecuteResultOnly[model.User](ctx, s.GetByUsernameB(opt))
}

func (s *Service) GetByUsernameB(opt GetByUsernameOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, opt.Tenant).
		SetPathParam(ParamUsername, opt.Username).
		SetURL(ApiUserByName)
	return core.NewTryRequest(s.Client, req)
}

// Create a user
func (s *Service) Create(ctx context.Context, body any) (*model.User, error) {
	return core.ExecuteResultOnly[model.User](ctx, s.CreateB(body))
}

func (s *Service) CreateB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiUsers)
	return core.NewTryRequest(s.Client, req)
}

// Update a user
func (s *Service) Update(ctx context.Context, target Target, body any) (*model.User, error) {
	return core.ExecuteResultOnly[model.User](ctx, s.UpdateB(target, body))
}

func (s *Service) UpdateB(target Target, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, target.ID).
		SetPathParam(ParamTenantId, target.Tenant).
		SetBody(body).
		SetURL(ApiUser)
	return core.NewTryRequest(s.Client, req)
}

// Delete a user
func (s *Service) Delete(ctx context.Context, target Target) error {
	return core.ExecuteNoResult(ctx, s.DeleteB(target))
}

func (s *Service) DeleteB(target Target) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, target.ID).
		SetPathParam(ParamTenantId, target.Tenant).
		SetURL(ApiUser)
	return core.NewTryRequest(s.Client, req)
}

// ListGroupsOptions to filter the user groups which contain a given user
type ListGroupsOptions struct {
	// Defaults to the current tenant
	Tenant string `url:"-"`

	UserID string `url:"-"`

	pagination.PaginationOptions
}

// List groups that contain a given user in the tenant
func (s *Service) ListGroupsWithUser(ctx context.Context, opt ListGroupsOptions) (*model.UserCollection, error) {
	return core.ExecuteResultOnly[model.UserCollection](ctx, s.ListGroupsWithUserB(opt))
}

func (s *Service) ListGroupsWithUserB(opt ListGroupsOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, opt.Tenant).
		SetPathParam(ParamId, opt.UserID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiUserGroupsWithUser)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}
