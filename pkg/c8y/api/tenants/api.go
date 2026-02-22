package tenants

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/currenttenant"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/systemoptions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/tenantoptions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/usagestatistics"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiTenants = "/tenant/tenants"
var ApiTenant = "/tenant/tenants/{id}"
var ApiTenantCurrent = "/tenant/currentTenant"
var ApiTenantApplications = "/tenant/tenants/{tenantID}/applications"

const ParamId = "id"
const ParamTenantId = "tenantID"

const ResultProperty = "tenants"
const ApplicationReferencesResultProperty = "references"

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

// TenantIterator provides iteration over tenants
type TenantIterator = pagination.Iterator[jsonmodels.Tenant]

// List tenants
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Tenant] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewTenant)
}

// ListAll returns an iterator for all tenants
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *TenantIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Tenant] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewTenant,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiTenants)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Create a tenant
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Tenant] {
	return core.Execute(ctx, s.createB(body), jsonmodels.NewTenant)
}

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenants)
	return core.NewTryRequest(s.Client, req, "")
}

// Get a tenant
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.Tenant] {
	return core.Execute(ctx, s.getB(ID), jsonmodels.NewTenant)
}

func (s *Service) getB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenant)
	return core.NewTryRequest(s.Client, req)
}

// Update a tenant
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.Tenant] {
	return core.Execute(ctx, s.updateB(ID, body), jsonmodels.NewTenant)
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
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
func (s *Service) Delete(ctx context.Context, ID string, opt DeleteOptions) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(ID, opt))
}

func (s *Service) deleteB(ID string, opt DeleteOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiTenant)
	return core.NewTryRequest(s.Client, req)
}

// ListApplicationReferencesOptions options for listing application references
type ListApplicationReferencesOptions struct {
	// Pagination options
	pagination.PaginationOptions
}

// ApplicationReferenceIterator provides iteration over application references
type ApplicationReferenceIterator = pagination.Iterator[jsonmodels.ApplicationReference]

// ListApplicationReferences retrieves all applications subscribed or owned by a specific tenant
// Note: Can only be called from the management tenant
func (s *Service) ListApplicationReferences(ctx context.Context, tenantID string, opt ListApplicationReferencesOptions) op.Result[jsonmodels.ApplicationReference] {
	return core.ExecuteCollection(ctx, s.listApplicationReferencesB(tenantID, opt), ApplicationReferencesResultProperty, types.ResponseFieldStatistics, jsonmodels.NewApplicationReference)
}

// ListAllApplicationReferences returns an iterator for all application references
func (s *Service) ListAllApplicationReferences(ctx context.Context, tenantID string, opts ListApplicationReferencesOptions) *ApplicationReferenceIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.ApplicationReference] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.ListApplicationReferences(ctx, tenantID, o)
		},
		jsonmodels.NewApplicationReference,
	)
}

func (s *Service) listApplicationReferencesB(tenantID string, opt ListApplicationReferencesOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantId, tenantID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiTenantApplications)
	return core.NewTryRequest(s.Client, req, ApplicationReferencesResultProperty)
}
