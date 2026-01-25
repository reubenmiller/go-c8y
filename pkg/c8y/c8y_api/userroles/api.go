package userroles

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/userroles/usergroups"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/userroles/users"
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

// Retrieve all user roles in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) (*model.UserRoleCollection, error) {
	return core.ExecuteResultOnly[model.UserRoleCollection](ctx, s.ListB(opt))
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
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
func (s *Service) Get(ctx context.Context, opt GetOption) (*model.UserRole, error) {
	return core.ExecuteResultOnly[model.UserRole](ctx, s.GetB(opt))
}

func (s *Service) GetB(opt GetOption) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamName, opt.Name).
		SetURL(ApiRole)
	return core.NewTryRequest(s.Client, req)
}
