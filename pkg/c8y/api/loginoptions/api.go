package loginoptions

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/loginoptions/accessmappings"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/loginoptions/inventoryaccessmappings"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiLoginOptions = "/tenant/loginOptions"
var ApiLoginOption = "/tenant/loginOptions/{id}"
var ApiLoginOptionRestrict = "/tenant/loginOptions/{typeOrId}/restrict"

var ParamID = "id"
var ParamTypeOrID = "typeOrId"

const ResultProperty = "loginOptions"

func NewService(s *core.Service) *Service {
	return &Service{
		Service:                 *s,
		AccessMappings:          accessmappings.NewService(s),
		InventoryAccessMappings: inventoryaccessmappings.NewService(s),
	}
}

// Service provides api to get/set/delete login options in Cumulocity, plus their access
// mappings, inventory access mappings, and access restriction.
type Service struct {
	core.Service

	// AccessMappings manages a login option's access mappings (applications/groups).
	AccessMappings *accessmappings.Service
	// InventoryAccessMappings manages a login option's inventory-role access mappings.
	InventoryAccessMappings *inventoryaccessmappings.Service
}

// RestrictOptions is the body for restricting access to a login option (PUT .../restrict).
type RestrictOptions struct {
	// OnlyManagementTenantAccess restricts the login option to the management tenant.
	OnlyManagementTenantAccess bool `json:"onlyManagementTenantAccess"`
}

// Restrict updates the access restriction of a login option (identified by type or id).
func (s *Service) Restrict(ctx context.Context, typeOrID string, opt RestrictOptions) op.Result[jsonmodels.LoginOption] {
	return core.Execute(ctx, s.restrictB(typeOrID, opt), jsonmodels.NewLoginOption)
}

func (s *Service) restrictB(typeOrID string, opt RestrictOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetPathParam(ParamTypeOrID, typeOrID).
		SetBody(opt).
		SetURL(ApiLoginOptionRestrict)
	return core.NewTryRequest(s.Client, req)
}

// ListOptions is generated from the OpenAPI spec — see zz_generated_options.go.
// (TenantID is now correctly typed string; it was previously a typo'd bool.)

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
		SetURL(ApiLoginOptions)
	return core.NewTryRequest(s.Client, req, ResultProperty).WithNoAuth()
}

// Get login option
func (s *Service) Get(ctx context.Context, typeOrID string) op.Result[jsonmodels.LoginOption] {
	return core.Execute(ctx, s.getB(typeOrID), jsonmodels.NewLoginOption)
}

func (s *Service) getB(typeOrID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, typeOrID).
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
		SetPathParam(ParamID, typeOrID).
		SetBody(body).
		SetURL(ApiLoginOption)
	return core.NewTryRequest(s.Client, req)
}

// Delete a specific login option in the tenant by a given type or ID
func (s *Service) Delete(ctx context.Context, typeOrID string) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(typeOrID)).IgnoreNotFound()
}

func (s *Service) deleteB(typeOrID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamID, typeOrID).
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
func (s *Service) UpdateAccess(ctx context.Context, opt UpdateAccessOptions, body LoginOptionTenantAccess) op.Result[jsonmodels.LoginOption] {
	return core.Execute(ctx, s.updateAccessB(opt, body), jsonmodels.NewLoginOption)
}

func (s *Service) updateAccessB(opt UpdateAccessOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, opt.TypeOrId).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, opt.TypeOrId).
		SetBody(body).
		SetURL(ApiLoginOption)
	return core.NewTryRequest(s.Client, req)
}
