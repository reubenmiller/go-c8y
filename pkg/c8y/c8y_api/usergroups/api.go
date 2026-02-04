package usergroups

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/users/currentuser"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/users/groups"
	"resty.dev/v3"
)

var ApiGroups = "/user/{tenantID}/groups"
var ApiGroup = "/user/{tenantID}/groups/{id}"
var ApiGroupsWithUser = "/user/{tenantID}/users/{id}/groups"
var ApiGroupByName = "/user/{tenantID}/groupByName/{groupName}"

var ParamId = "id"
var ParamUsername = "username"
var ParamTenantId = "tenantID"
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
	return core.ExecuteReturnCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewUserGroup)
}

// ListAll returns an iterator for all user groups
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *UserGroupIterator {
	return pagination.Paginate(ctx, opts.PaginationOptions, func() op.Result[jsonmodels.UserGroup] {
		return s.List(ctx, opts)
	}, jsonmodels.NewUserGroup)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, opt.Tenant).
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
	return core.ExecuteReturnResult(ctx, s.getB(opt), jsonmodels.NewUserGroup)
}

func (s *Service) getB(target Target) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, target.Tenant).
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
	return core.ExecuteReturnResult(ctx, s.getByNameB(opt), jsonmodels.NewUserGroup)
}

func (s *Service) getByNameB(opt GetByNameOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, opt.Tenant).
		SetPathParam(ParamGroupName, opt.GroupName).
		SetURL(ApiGroupByName)
	return core.NewTryRequest(s.Client, req)
}

// Create a user group
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.UserGroup] {
	return core.ExecuteReturnResult(ctx, s.createB(body), jsonmodels.NewUserGroup)
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
	return core.ExecuteReturnResult(ctx, s.updateB(opt, body), jsonmodels.NewUserGroup)
}

func (s *Service) updateB(opt UpdateOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, opt.Tenant).
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
	return core.ExecuteNoResult(ctx, s.deleteB(opt))
}

func (s *Service) deleteB(opt DeleteOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamTenantId, opt.Tenant).
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
	return core.ExecuteReturnCollection(ctx, s.listByUserB(opt), "references", "group", jsonmodels.NewUserGroup)
}

func (s *Service) listByUserB(opt ListByUserOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, opt.Tenant).
		SetPathParam(ParamId, opt.UserID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiGroupsWithUser)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}
