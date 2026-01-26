package tenantoptions

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiTenantOptions = "/tenant/options"
var ApiTenantOption = "/tenant/options/{category}/{key}"
var ApiTenantOptionEditable = "/tenant/options/{category}/{key}/editable"
var ApiTenantOptionByCategory = "/tenant/options/{category}"

const ParamKey = "key"
const ParamCategory = "category"

const ResultProperty = "options"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service api to interact with tenant options
// type Service core.Service
type Service struct {
	core.Service
}

// ListOptions tenant options filter
type ListOptions struct {
	// Pagination options
	pagination.PaginationOptions
}

// List tenant options
func (s *Service) List(ctx context.Context, opt ListOptions) (*model.TenantOptionCollection, error) {
	return core.ExecuteResultOnly[model.TenantOptionCollection](ctx, s.ListB(opt))
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiTenantOptions)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Create a tenant option
func (s *Service) Create(ctx context.Context, body any) (*model.TenantOption, error) {
	return core.ExecuteResultOnly[model.TenantOption](ctx, s.CreateB(body))
}

func (s *Service) CreateB(body any) *core.TryRequest {
	req := s.Service.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenantOptions)
	return core.NewTryRequest(s.Client, req, "")
}

type GetOption struct {
	Key      string `url:"-"`
	Category string `url:"-"`
}

// Get a tenant option
func (s *Service) Get(ctx context.Context, opt GetOption) (*model.TenantOption, error) {
	return core.ExecuteResultOnly[model.TenantOption](ctx, s.GetB(opt))
}

func (s *Service) GetB(opt GetOption) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamCategory, opt.Category).
		SetPathParam(ParamKey, opt.Key).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenantOption)
	return core.NewTryRequest(s.Client, req)
}

type UpdateOption struct {
	Category string `url:"-"`
	Key      string `url:"-"`

	Body any `url:"-"`
}

// Update a tenant option
func (s *Service) Update(ctx context.Context, opt UpdateOption) (*model.TenantOption, error) {
	return core.ExecuteResultOnly[model.TenantOption](ctx, s.UpdateB(opt))
}

func (s *Service) UpdateB(opt UpdateOption) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamCategory, opt.Category).
		SetPathParam(ParamKey, opt.Key).
		SetBody(opt.Body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenantOption)
	return core.NewTryRequest(s.Client, req)
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
func (s *Service) UpdateEditableFlag(ctx context.Context, opt UpdateEditableFlagOption) (*model.TenantOption, error) {
	return core.ExecuteResultOnly[model.TenantOption](ctx, s.UpdateEditableFlagB(opt))
}

func (s *Service) UpdateEditableFlagB(opt UpdateEditableFlagOption) *core.TryRequest {
	body := map[string]any{
		"editable": opt.Editable,
	}
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamCategory, opt.Category).
		SetPathParam(ParamKey, opt.Key).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenantOptionEditable)
	return core.NewTryRequest(s.Client, req)
}

// DeleteOptions options to delete a tenant
type DeleteOptions struct {
	Key      string `url:"-"`
	Category string `url:"-"`
}

// Delete a tenant option
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) error {
	return core.ExecuteNoResult(ctx, s.DeleteB(opt))
}

func (s *Service) DeleteB(opt DeleteOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamCategory, opt.Category).
		SetPathParam(ParamKey, opt.Key).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiTenantOption)
	return core.NewTryRequest(s.Client, req)
}

// ListByCategoryOptions tenant options filter
type ListByCategoryOptions struct {
	Category string `url:"-"`

	// Pagination options
	pagination.PaginationOptions
}

// List tenant options by category
func (s *Service) ListByCategory(ctx context.Context, opt ListByCategoryOptions) (map[string]string, error) {
	return core.ExecuteResultMap[string, string](ctx, s.ListByCategoryB(opt))
}

func (s *Service) ListByCategoryB(opt ListByCategoryOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamCategory, opt.Category).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiTenantOptionByCategory)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type UpdateByCategoryOption struct {
	Category string `url:"-"`

	Body map[string]string `url:"-"`
}

// Update a tenant option
func (s *Service) UpdateByCategory(ctx context.Context, opt UpdateByCategoryOption) (map[string]string, error) {
	return core.ExecuteResultMap[string, string](ctx, s.UpdateByCategoryOptionB(opt))
}

func (s *Service) UpdateByCategoryOptionB(opt UpdateByCategoryOption) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamCategory, opt.Category).
		SetBody(opt.Body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenantOptionByCategory)
	return core.NewTryRequest(s.Client, req)
}
