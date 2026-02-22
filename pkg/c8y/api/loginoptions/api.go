package loginoptions

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiLoginOptions = "/tenant/loginOptions"
var ApiLoginOption = "/tenant/loginOptions/{id}"

var ParamId = "id"

const ResultProperty = "loginOptions"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service provides api to get/set/delete events in Cumulocity
type Service struct {
	core.Service
}

// ListOptions to filter the login options by
type ListOptions struct {
	// If this is set to true, the management tenant login options will be returned
	Management bool `url:"management,omitempty"`

	// Unique identifier of a Cumulocity tenant
	TenantID bool `url:"tenantId,omitempty"`

	// Pagination options
	pagination.PaginationOptions
}

// LoginOptionIterator provides iteration over login options
type LoginOptionIterator = pagination.Iterator[jsonmodels.LoginOption]

// Retrieve all login options available in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.LoginOption] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewLoginOption)
}

// ListAll returns an iterator for all login options
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *LoginOptionIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.LoginOption] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewLoginOption,
	)
}

func (s *Service) listB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiLoginOptions)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Retrieve all login options available in the tenant without using credentials
func (s *Service) ListNoAuth(ctx context.Context, opt ListOptions) op.Result[jsonmodels.LoginOption] {
	return core.ExecuteCollection(ctx, s.listNoAuthB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewLoginOption)
}

func (s *Service) listNoAuthB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		Funcs(core.NoAuthorization()).
		SetURL(ApiLoginOptions)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get an event
func (s *Service) Get(ctx context.Context, typeOrID string) op.Result[jsonmodels.LoginOption] {
	return core.Execute(ctx, s.getB(typeOrID), jsonmodels.NewLoginOption)
}

func (s *Service) getB(typeOrID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, typeOrID).
		SetURL(ApiLoginOption)
	return core.NewTryRequest(s.Client, req)
}

// Create a login option
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.LoginOption] {
	return core.Execute(ctx, s.createB(body), jsonmodels.NewLoginOption)
}

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiLoginOptions)
	return core.NewTryRequest(s.Client, req)
}

// Update a login option
func (s *Service) Update(ctx context.Context, typeOrID string, body any) op.Result[jsonmodels.LoginOption] {
	return core.Execute(ctx, s.updateB(typeOrID, body), jsonmodels.NewLoginOption)
}

func (s *Service) updateB(typeOrID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, typeOrID).
		SetBody(body).
		SetURL(ApiLoginOption)
	return core.NewTryRequest(s.Client, req)
}

// Delete a specific login option in the tenant by a given type or ID
func (s *Service) Delete(ctx context.Context, typeOrID string) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(typeOrID))
}

func (s *Service) deleteB(typeOrID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, typeOrID).
		SetURL(ApiLoginOption)
	return core.NewTryRequest(s.Client, req)
}

type UpdateAccessOptions struct {
	// The type or ID of the login option. The type's value is case insensitive and can be OAUTH2, OAUTH2_INTERNAL or BASIC
	TypeOrId string `url:"-"`

	// Unique identifier of a Cumulocity tenant
	TargetTenant string `url:"targetTenant,omitempty"`
}

type LoginOptionTenantAccess struct {
	// Indicates whether the configuration is only accessible to the management tenant
	OnlyManagementTenantAccess bool `json:"onlyManagementTenantAccess"`
}

// Update the tenant's access to the authentication configuration.
// TODO: This function signature is awkward
func (s *Service) UpdateAccess(ctx context.Context, opt UpdateAccessOptions, body LoginOptionTenantAccess) op.Result[jsonmodels.LoginOption] {
	return core.Execute(ctx, s.updateAccessB(opt, body), jsonmodels.NewLoginOption)
}

func (s *Service) updateAccessB(opt UpdateAccessOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, opt.TypeOrId).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, opt.TypeOrId).
		SetBody(body).
		SetURL(ApiLoginOption)
	return core.NewTryRequest(s.Client, req)
}
