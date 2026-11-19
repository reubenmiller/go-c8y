package tenantoptions

import (
	"context"
	"encoding/json"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiTenantOptions = "/tenant/options"
var ApiTenantOption = "/tenant/options/{category}/{key}"
var ApiTenantOptionEditable = "/tenant/options/{category}/{key}/editable"
var ApiTenantOptionByCategory = "/tenant/options/{category}"

const ParamKey = "key"
const ParamCategory = "category"

const ResultProperty = "options"

func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc: *s,
	}
}

// Service api to interact with tenant options
// type Service httpcore.Service
type Service struct {
	svc httpcore.Service
}

// ListOptions tenant options filter
type ListOptions struct {
	// Pagination options
	pagination.PaginationOptions
}

// TenantOptionIterator provides iteration over tenant options
type TenantOptionIterator = pagination.Iterator[jsonmodels.TenantOption]

// List tenant options
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.TenantOption] {
	return httpcore.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewTenantOption)
}

// ListAll returns an iterator for all tenant options
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *TenantOptionIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.TenantOption] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewTenantOption,
	)
}

func (s *Service) listB(opt ListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(ApiTenantOptions)
	return httpcore.NewTryRequest(s.svc.Client, req, ResultProperty)
}

// Create a tenant option
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.TenantOption] {
	return httpcore.Execute(ctx, s.createB(body), jsonmodels.NewTenantOption)
}

func (s *Service) createB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenantOptions)
	return httpcore.NewTryRequest(s.svc.Client, req, "")
}

type GetOption struct {
	Key      string `url:"-"`
	Category string `url:"-"`
}

// Get a tenant option
func (s *Service) Get(ctx context.Context, opt GetOption) op.Result[jsonmodels.TenantOption] {
	return httpcore.Execute(ctx, s.getB(opt), jsonmodels.NewTenantOption)
}

func (s *Service) getB(opt GetOption) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamCategory, opt.Category).
		SetPathParam(ParamKey, opt.Key).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenantOption)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type UpdateOption struct {
	Category string `url:"-"`
	Key      string `url:"-"`

	Body any `url:"-"`
}

// Update a tenant option
func (s *Service) Update(ctx context.Context, opt UpdateOption) op.Result[jsonmodels.TenantOption] {
	return httpcore.Execute(ctx, s.updateB(opt), jsonmodels.NewTenantOption)
}

func (s *Service) updateB(opt UpdateOption) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamCategory, opt.Category).
		SetPathParam(ParamKey, opt.Key).
		SetBody(opt.Body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenantOption)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type UpdateEditableFlagOption struct {
	Category string `url:"-"`
	Key      string `url:"-"`

	// Unique identifier of a Cumulocity tenant.
	TargetTenant string `url:"targetTenant,omitempty"`

	// Indicates if option can be edited
	Editable bool `url:"-"`
}

// Updates the editable flag of a specific option (by a given category and key) on target tenant which determines if the option can be edited
func (s *Service) UpdateEditableFlag(ctx context.Context, opt UpdateEditableFlagOption) op.Result[jsonmodels.TenantOption] {
	return httpcore.Execute(ctx, s.updateEditableFlagB(opt), jsonmodels.NewTenantOption)
}

func (s *Service) updateEditableFlagB(opt UpdateEditableFlagOption) *httpcore.TryRequest {
	body := map[string]any{
		"editable": opt.Editable,
	}
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamCategory, opt.Category).
		SetPathParam(ParamKey, opt.Key).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenantOptionEditable)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// DeleteOptions options to delete a tenant
type DeleteOptions struct {
	Key      string `url:"-"`
	Category string `url:"-"`
}

// Delete a tenant option
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.deleteB(opt)).IgnoreNotFound()
}

func (s *Service) deleteB(opt DeleteOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamCategory, opt.Category).
		SetPathParam(ParamKey, opt.Key).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(ApiTenantOption)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// ListByCategoryOptions tenant options filter
type ListByCategoryOptions struct {
	Category string `url:"-"`

	// Pagination options
	pagination.PaginationOptions
}

// List tenant options by category
func (s *Service) ListByCategory(ctx context.Context, opt ListByCategoryOptions) op.Result[map[string]string] {
	return httpcore.Execute(ctx, s.listByCategoryB(opt), func(b []byte) map[string]string {
		data := make(map[string]string)
		_ = json.Unmarshal(b, &data)
		return data
	})
}

func (s *Service) listByCategoryB(opt ListByCategoryOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamCategory, opt.Category).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(ApiTenantOptionByCategory)
	return httpcore.NewTryRequest(s.svc.Client, req, ResultProperty)
}

type UpdateByCategoryOption struct {
	Category string `url:"-"`

	Body map[string]string `url:"-"`
}

// Update a tenant option
func (s *Service) UpdateByCategory(ctx context.Context, opt UpdateByCategoryOption) op.Result[map[string]string] {
	return httpcore.Execute(ctx, s.updateByCategoryOptionB(opt), func(b []byte) map[string]string {
		data := make(map[string]string)
		_ = json.Unmarshal(b, &data)
		return data
	})
}

func (s *Service) updateByCategoryOptionB(opt UpdateByCategoryOption) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamCategory, opt.Category).
		SetBody(opt.Body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenantOptionByCategory)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
