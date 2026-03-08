package usergroups

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/users/currentuser"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/users/groups"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiGroups = "/user/{tenantId}/groups"
var ApiGroup = "/user/{tenantId}/groups/{id}"
var ApiGroupsWithUser = "/user/{tenantId}/users/{id}/groups"
var ApiGroupByName = "/user/{tenantId}/groupByName/{groupName}"

var ParamId = "id"
var ParamUsername = "username"
var ParamGroupName = "groupName"

const ResultProperty = "groups"

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

// ListOptions to filter the groups by
type ListOptions struct {
	// Defaults to the current tenant
	Tenant string `url:"-"`

	pagination.PaginationOptions
}

// UserGroupIterator provides iteration over user groups
type UserGroupIterator = pagination.Iterator[jsonmodels.UserGroup]

// Retrieve all groups in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.UserGroup] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewUserGroup)
}

// ListAll returns an iterator for all user groups
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *UserGroupIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.UserGroup] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewUserGroup,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiGroups)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type Target struct {
	Tenant string `url:"-"`

	ID string `url:"-"`
}

// Get a user group
func (s *Service) Get(ctx context.Context, opt Target) op.Result[jsonmodels.UserGroup] {
	return core.Execute(ctx, s.getB(opt), jsonmodels.NewUserGroup)
}

func (s *Service) getB(target Target) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, target.Tenant).
		SetPathParam(ParamId, target.ID).
		SetURL(ApiGroup)
	return core.NewTryRequest(s.Client, req)
}

type GetByNameOptions struct {
	GroupName string
	Tenant    string
}

// Get a group by name
func (s *Service) GetByName(ctx context.Context, opt GetByNameOptions) op.Result[jsonmodels.UserGroup] {
	return core.Execute(ctx, s.getByNameB(opt), jsonmodels.NewUserGroup)
}

func (s *Service) getByNameB(opt GetByNameOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetPathParam(ParamGroupName, opt.GroupName).
		SetURL(ApiGroupByName)
	return core.NewTryRequest(s.Client, req)
}

// Create a user group
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.UserGroup] {
	return core.Execute(ctx, s.createB(body), jsonmodels.NewUserGroup)
}

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiGroups)
	return core.NewTryRequest(s.Client, req)
}

type UpdateOptions struct {
	Target

	ForceLogout bool `url:"forceLogout,omitzero"`
}

// Update a user group
func (s *Service) Update(ctx context.Context, opt UpdateOptions, body any) op.Result[jsonmodels.UserGroup] {
	return core.Execute(ctx, s.updateB(opt, body), jsonmodels.NewUserGroup)
}

func (s *Service) updateB(opt UpdateOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetPathParam(ParamId, opt.ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetBody(body).
		SetURL(ApiGroup)
	return core.NewTryRequest(s.Client, req)
}

type DeleteOptions struct {
	Target

	ForceLogout bool `url:"forceLogout,omitzero"`
}

// Delete a user group
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(opt))
}

func (s *Service) deleteB(opt DeleteOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetPathParam(ParamId, opt.ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiGroup)
	return core.NewTryRequest(s.Client, req)
}

// ListByUserOptions to filter the user groups which contain a given user
type ListByUserOptions struct {
	// Defaults to the current tenant
	Tenant string `url:"-"`

	UserID string `url:"-"`

	pagination.PaginationOptions
}

// List groups that contain a given user in the tenant
func (s *Service) ListByUser(ctx context.Context, opt ListByUserOptions) op.Result[jsonmodels.UserGroup] {
	return core.ExecuteCollection(ctx, s.listByUserB(opt), "references.#.group", types.ResponseFieldStatistics, jsonmodels.NewUserGroup)
}

func (s *Service) listByUserB(opt ListByUserOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetPathParam(ParamId, opt.UserID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiGroupsWithUser)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}
