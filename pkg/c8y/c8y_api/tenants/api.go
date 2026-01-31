package tenants

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/tenants/currenttenant"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/tenants/systemoptions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/tenants/tenantoptions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/tenants/usagestatistics"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiTenants = "/tenant/tenants"
var ApiTenant = "/tenant/tenants/{id}"
var ApiTenantCurrent = "/tenant/currentTenant"

const ParamId = "id"

const ResultProperty = "tenants"

func NewService(s *core.Service) *Service {
	return &Service{
		Service:         *s,
		UsageStatistics: usagestatistics.NewService(s),
		Current:         currenttenant.NewService(s),
		Options:         tenantoptions.NewService(s),
		SystemOptions:   systemoptions.NewService(s),
	}
}

// Service inventory api to interact with managed objects
// type Service core.Service
type Service struct {
	core.Service
	UsageStatistics *usagestatistics.Service
	Current         *currenttenant.Service
	Options         *tenantoptions.Service
	SystemOptions   *systemoptions.Service
}

// ListOptions tenant filter options
type ListOptions struct {
	// Company name associated with the Cumulocity tenant
	Company string `url:"company,omitempty"`

	// Domain name of the Cumulocity tenant
	Domain string `url:"domain,omitempty"`

	// Identifier of the Cumulocity tenant's parent. Works only for requests sent with management tenant credentials
	Parent string `url:"parent,omitempty"`

	// Pagination options
	pagination.PaginationOptions
}

// List tenants
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Tenant] {
	return core.ExecuteReturnCollection(ctx, s.ListB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewTenant)
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiTenants)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Create a tenant
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Tenant] {
	return core.ExecuteReturnResult(ctx, s.CreateB(body), jsonmodels.NewTenant)
}

func (s *Service) CreateB(body any) *core.TryRequest {
	req := s.Service.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenants)
	return core.NewTryRequest(s.Client, req, "")
}

// Get a tenant
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.Tenant] {
	return core.ExecuteReturnResult(ctx, s.GetB(ID), jsonmodels.NewTenant)
}

func (s *Service) GetB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenant)
	return core.NewTryRequest(s.Client, req)
}

// Update a tenant
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.Tenant] {
	return core.ExecuteReturnResult(ctx, s.UpdateB(ID, body), jsonmodels.NewTenant)
}

func (s *Service) UpdateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenant)
	return core.NewTryRequest(s.Client, req)
}

// DeleteOptions options to delete a tenant
type DeleteOptions struct{}

// Delete a tenant
//
// Important: Deleting a subtenant cannot be reverted. For security reasons, it is therefore only available in the management tenant. You cannot delete tenants from any tenant but the management tenant.
// Administrators in Enterprise Tenants are only allowed to suspend active subtenants, but not to delete them.
func (s *Service) Delete(ctx context.Context, ID string, opt DeleteOptions) error {
	return core.ExecuteNoResult(ctx, s.DeleteB(ID, opt))
}

func (s *Service) DeleteB(ID string, opt DeleteOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiTenant)
	return core.NewTryRequest(s.Client, req)
}
