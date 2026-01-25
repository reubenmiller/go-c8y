package loginoptions

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
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
}

// Retrieve all login options available in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) (*model.LoginOptionCollection, error) {
	return core.ExecuteResultOnly[model.LoginOptionCollection](ctx, s.ListB(opt))
}

func (s *Service) ListB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiLoginOptions)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Retrieve all login options available in the tenant without using credentials
func (s *Service) ListNoAuth(ctx context.Context, opt ListOptions) (*model.LoginOptionCollection, error) {
	return core.ExecuteResultOnly[model.LoginOptionCollection](ctx, s.ListNoAuthB(opt))
}

func (s *Service) ListNoAuthB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		Funcs(core.NoAuthorization()).
		SetURL(ApiLoginOptions)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get an event
func (s *Service) Get(ctx context.Context, typeOrID string) (*model.LoginOption, error) {
	return core.ExecuteResultOnly[model.LoginOption](ctx, s.GetB(typeOrID))
}

func (s *Service) GetB(typeOrID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, typeOrID).
		SetURL(ApiLoginOption)
	return core.NewTryRequest(s.Client, req)
}

// Create a login option
func (s *Service) Create(ctx context.Context, body any) (*model.LoginOption, error) {
	return core.ExecuteResultOnly[model.LoginOption](ctx, s.CreateB(body))
}

func (s *Service) CreateB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiLoginOptions)
	return core.NewTryRequest(s.Client, req)
}

// Update a login option
func (s *Service) Update(ctx context.Context, typeOrID string, body any) (*model.LoginOption, error) {
	return core.ExecuteResultOnly[model.LoginOption](ctx, s.UpdateB(typeOrID, body))
}

func (s *Service) UpdateB(typeOrID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, typeOrID).
		SetBody(body).
		SetURL(ApiLoginOption)
	return core.NewTryRequest(s.Client, req)
}

// Delete a specific login option in the tenant by a given type or ID
func (s *Service) Delete(ctx context.Context, typeOrID string) error {
	return core.ExecuteNoResult(ctx, s.DeleteB(typeOrID))
}

func (s *Service) DeleteB(typeOrID string) *core.TryRequest {
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
func (s *Service) UpdateAccess(ctx context.Context, opt UpdateAccessOptions, body LoginOptionTenantAccess) (*model.LoginOption, error) {
	return core.ExecuteResultOnly[model.LoginOption](ctx, s.UpdateAccessB(opt, body))
}

func (s *Service) UpdateAccessB(opt UpdateAccessOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, opt.TypeOrId).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, opt.TypeOrId).
		SetBody(body).
		SetURL(ApiLoginOption)
	return core.NewTryRequest(s.Client, req)
}
