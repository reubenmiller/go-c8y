package users

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/users/currentuser"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/users/groups"
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

// UserIterator provides iteration over users
type UserIterator = pagination.Iterator[jsonmodels.User]

// Retrieve all users in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.User] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewUser)
}

// ListAll returns an iterator for all users
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
		SetPathParam(ParamTenantId, opt.Tenant).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiUsers)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type Target struct {
	ID     string
	Tenant string
}

type GetOptions Target

// Get a user
func (s *Service) Get(ctx context.Context, opt GetOptions) op.Result[jsonmodels.User] {
	return core.Execute(ctx, s.getB(opt), jsonmodels.NewUser)
}

func (s *Service) getB(opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, opt.Tenant).
		SetPathParam(ParamId, opt.ID).
		SetURL(ApiUser)
	return core.NewTryRequest(s.Client, req)
}

type GetByUsernameOptions struct {
	Username string
	Tenant   string
}

// Get a user by username
func (s *Service) GetByUsername(ctx context.Context, opt GetByUsernameOptions) op.Result[jsonmodels.User] {
	return core.Execute(ctx, s.getByUsernameB(opt), jsonmodels.NewUser)
}

func (s *Service) getByUsernameB(opt GetByUsernameOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, opt.Tenant).
		SetPathParam(ParamUsername, opt.Username).
		SetURL(ApiUserByName)
	return core.NewTryRequest(s.Client, req)
}

// Create a user
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.User] {
	return core.Execute(ctx, s.createB(body), jsonmodels.NewUser)
}

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiUsers)
	return core.NewTryRequest(s.Client, req)
}

type UpdateOptions Target

// Update a user
func (s *Service) Update(ctx context.Context, opt UpdateOptions, body any) op.Result[jsonmodels.User] {
	return core.Execute(ctx, s.updateB(opt, body), jsonmodels.NewUser)
}

func (s *Service) updateB(opt UpdateOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, opt.ID).
		SetPathParam(ParamTenantId, opt.Tenant).
		SetBody(body).
		SetURL(ApiUser)
	return core.NewTryRequest(s.Client, req)
}

type DeleteOptions Target

// Delete a user
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(opt))
}

func (s *Service) deleteB(opt DeleteOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, opt.ID).
		SetPathParam(ParamTenantId, opt.Tenant).
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
func (s *Service) ListGroupsWithUser(ctx context.Context, opt ListGroupsOptions) op.Result[jsonmodels.UserGroup] {
	return core.ExecuteCollection(ctx, s.listGroupsWithUserB(opt), "references", "group", jsonmodels.NewUserGroup)
}

func (s *Service) listGroupsWithUserB(opt ListGroupsOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, opt.Tenant).
		SetPathParam(ParamId, opt.UserID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiUserGroupsWithUser)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}
