package tenants

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/currenttenant"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/devicestatistics"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/logintokens"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/restrictedroles"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/systemoptions"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/tenantoptions"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/usagestatistics"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/usagestatisticsfiles"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var apiTenants = "/tenant/tenants"
var apiTenant = "/tenant/tenants/{id}"
var apiTenantCurrent = "/tenant/currentTenant"
var apiTenantApplications = "/tenant/tenants/{tenantId}/applications"
var apiTenantApplication = "/tenant/tenants/{tenantId}/applications/{applicationId}"
var apiTenantTFA = "/tenant/tenants/{tenantId}/tfa"

const paramID = "id"
const paramApplicationID = "applicationId"

const resultProperty = "tenants"
const ApplicationReferencesResultProperty = "references"

func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc:                  *s,
		UsageStatistics:      usagestatistics.NewService(s),
		UsageStatisticsFiles: usagestatisticsfiles.NewService(s),
		Current:              currenttenant.NewService(s),
		Options:              tenantoptions.NewService(s),
		SystemOptions:        systemoptions.NewService(s),
		RestrictedRoles:      restrictedroles.NewService(s),
		DeviceStatistics:     devicestatistics.NewService(s),
		LoginTokens:          logintokens.NewService(s),
	}
}

// Service inventory api to interact with managed objects
// type Service httpcore.Service
type Service struct {
	svc             httpcore.Service
	UsageStatistics *usagestatistics.Service
	// UsageStatisticsFiles is the generated usage statistics files
	// sub-resource (/tenant/statistics/files).
	UsageStatisticsFiles *usagestatisticsfiles.Service
	Current              *currenttenant.Service
	Options              *tenantoptions.Service
	SystemOptions        *systemoptions.Service
	RestrictedRoles      *restrictedroles.Service

	// DeviceStatistics is the device statistics sub-resource
	// (/tenant/statistics/device).
	DeviceStatistics *devicestatistics.Service

	// LoginTokens is the login token sub-resource (/tenant/oauth/token).
	LoginTokens *logintokens.Service
}

// ListOptions is generated from the OpenAPI spec — see zz_generated_options.go.

// TenantIterator provides iteration over tenants
type TenantIterator = pagination.Iterator[jsonmodels.Tenant]

// List tenants
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Tenant] {
	return httpcore.ExecuteCollection(ctx, s.listB(opt), resultProperty, types.ResponseFieldStatistics, jsonmodels.NewTenant)
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

func (s *Service) listB(opt ListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiTenants)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

// Create a tenant
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Tenant] {
	return httpcore.Execute(ctx, s.createB(body), jsonmodels.NewTenant)
}

func (s *Service) createB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiTenants)
	return httpcore.NewTryRequest(s.svc.Client, req, "")
}

// Get a tenant
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.Tenant] {
	return httpcore.Execute(ctx, s.getB(ID), jsonmodels.NewTenant)
}

func (s *Service) getB(ID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(paramID, ID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiTenant)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Update a tenant
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.Tenant] {
	return httpcore.Execute(ctx, s.updateB(ID, body), jsonmodels.NewTenant)
}

func (s *Service) updateB(ID string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(paramID, ID).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiTenant)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// DeleteOptions options to delete a tenant
type DeleteOptions struct{}

// Delete a tenant
//
// Important: Deleting a subtenant cannot be reverted. For security reasons, it is therefore only available in the management tenant. You cannot delete tenants from any tenant but the management tenant.
// Administrators in Enterprise Tenants are only allowed to suspend active subtenants, but not to delete them.
func (s *Service) Delete(ctx context.Context, ID string, opt DeleteOptions) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.deleteB(ID, opt)).IgnoreNotFound()
}

func (s *Service) deleteB(ID string, opt DeleteOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(paramID, ID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiTenant)
	return httpcore.NewTryRequest(s.svc.Client, req)
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
	return httpcore.ExecuteCollection(ctx, s.listApplicationReferencesB(tenantID, opt), ApplicationReferencesResultProperty, types.ResponseFieldStatistics, jsonmodels.NewApplicationReference)
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

func (s *Service) listApplicationReferencesB(tenantID string, opt ListApplicationReferencesOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, tenantID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiTenantApplications)
	return httpcore.NewTryRequest(s.svc.Client, req, ApplicationReferencesResultProperty)
}

// SubscribeApplication subscribes (enables) an application on a tenant by POSTing
// an application reference body to /tenant/tenants/{tenantId}/applications. The
// body identifies the application by id or self link, e.g.
// {"application":{"id":"<id>"}}; the CLI resolves the application reference and
// sets application.id before calling this. Returns the created application
// reference as-is, matching the v1 enableApplication command's
// applicationReference output.
// Note: Can only be called from the management tenant.
func (s *Service) SubscribeApplication(ctx context.Context, tenantID string, body any) op.Result[jsonmodels.ApplicationReference] {
	return httpcore.Execute(ctx, s.subscribeApplicationB(tenantID, body), jsonmodels.NewApplicationReference)
}

func (s *Service) subscribeApplicationB(tenantID string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(core.PathParamTenantID, tenantID).
		SetBody(body).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiTenantApplications)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// UnsubscribeApplication unsubscribes (disables) an application from a tenant.
// The applicationID must already be resolved (the CLI resolves a name reference
// to an id first). Returns no content; a 404 (not subscribed) is ignored.
// Note: Can only be called from the management tenant.
func (s *Service) UnsubscribeApplication(ctx context.Context, tenantID string, applicationID string) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.unsubscribeApplicationB(tenantID, applicationID)).IgnoreNotFound()
}

func (s *Service) unsubscribeApplicationB(tenantID string, applicationID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(core.PathParamTenantID, tenantID).
		SetPathParam(paramApplicationID, applicationID).
		SetURL(apiTenantApplication)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// GetTFA retrieves the two-factor authentication settings of a specific tenant.
// Pass an empty tenantID to use the current context tenant.
func (s *Service) GetTFA(ctx context.Context, tenantID string) op.Result[jsonmodels.TenantTFA] {
	return httpcore.Execute(ctx, s.getTFAB(tenantID), jsonmodels.NewTenantTFA)
}

func (s *Service) getTFAB(tenantID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(core.PathParamTenantID, tenantID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiTenantTFA)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// UpdateTFA updates the two-factor authentication settings of a specific tenant.
// Pass an empty tenantID to use the current context tenant.
// The body should contain a "strategy" field with a value of "SMS" or "TOTP".
func (s *Service) UpdateTFA(ctx context.Context, tenantID string, body any) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.updateTFAB(tenantID, body))
}

func (s *Service) updateTFAB(tenantID string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(core.PathParamTenantID, tenantID).
		SetBody(body).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiTenantTFA)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
