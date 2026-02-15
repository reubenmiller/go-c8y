package userroles

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/userroles/usergroups"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/userroles/users"
	"resty.dev/v3"
)

var ApiRoles = "/user/roles"
var ApiRole = "/user/roles/{name}"

var ParamName = "name"

const ResultProperty = "roles"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
		Groups:  usergroups.NewService(s),
		Users:   users.NewService(s),
	}
}

// Service provides api to manage user roles
type Service struct {
	core.Service

	Groups *usergroups.Service
	Users  *users.Service
}

// ListOptions to filter the user roles by
type ListOptions struct {
	pagination.PaginationOptions
}

// RoleIterator provides iteration over roles
type RoleIterator = pagination.Iterator[jsonmodels.Role]

// Retrieve all user roles in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Role] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewRole)
}

// ListAll returns an iterator for all user roles
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *RoleIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Role] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewRole,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiRoles)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type GetOption struct {
	Name string `url:"-"`
}

// Get a user role
func (s *Service) Get(ctx context.Context, opt GetOption) op.Result[jsonmodels.Role] {
	return core.Execute(ctx, s.getB(opt), jsonmodels.NewRole)
}

func (s *Service) getB(opt GetOption) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamName, opt.Name).
		SetURL(ApiRole)
	return core.NewTryRequest(s.Client, req)
}
