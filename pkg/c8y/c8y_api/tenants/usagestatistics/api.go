package usagestatistics

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiTenantStatistics = "/tenant/statistics"
var ApiTenantStatisticsSummary = "/tenant/statistics/summary"
var ApiTenantStatisticsAllTenantsSummary = "/tenant/statistics/allTenantsSummary"

const ParamId = "id"
const ParamChild = "child"

const ResultProperty = "usageStatistics"

// Service
type Service core.Service

func NewService(common *core.Service) *Service {
	return (*Service)(common)
}

// ListOptions
type ListOptions struct {
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// Pagination options
	pagination.PaginationOptions
}

// TenantUsageStatisticsIterator provides iteration over tenant usage statistics
type TenantUsageStatisticsIterator = pagination.Iterator[jsonmodels.TenantUsageStatistics]

// List tenant statistics
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.TenantUsageStatistics] {
	return core.ExecuteReturnCollection(ctx, s.ListB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewTenantUsageStatistics)
}

// ListAll returns an iterator for all tenant usage statistics
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *TenantUsageStatisticsIterator {
	if opts.PageSize == 0 {
		opts.PageSize = 2000
	}
	return pagination.Paginate(ctx, func(page int) op.Result[jsonmodels.TenantUsageStatistics] {
		opts.CurrentPage = page
		return s.List(ctx, opts)
	}, jsonmodels.NewTenantUsageStatistics, opts.GetMaxItems())
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiTenantStatistics)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// ListSummaryOptions
type ListSummaryOptions struct {
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	Tenant string `url:"tenant,omitempty,omitzero"`
}

// List tenant statistics
func (s *Service) ListSummary(ctx context.Context, opt ListSummaryOptions) op.Result[jsonmodels.TenantUsageStatistics] {
	return core.ExecuteReturnResult(ctx, s.ListSummaryB(opt), jsonmodels.NewTenantUsageStatistics)
}

func (s *Service) ListSummaryB(opt ListSummaryOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiTenantStatisticsSummary)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// ListSummaryAllTenantsOptions
type ListSummaryAllTenantsOptions struct {
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	DateTo time.Time `url:"dateTo,omitempty,omitzero"`
}

// // List usage statistics of all tenants
func (s *Service) ListSummaryAllTenants(ctx context.Context, opt ListSummaryAllTenantsOptions) op.Result[jsonmodels.TenantUsageStatisticsSummary] {
	return core.ExecuteReturnCollection(ctx, s.ListSummaryAllTenantsB(opt), "", "", jsonmodels.NewTenantUsageStatisticsSummary)
}

func (s *Service) ListSummaryAllTenantsB(opt ListSummaryAllTenantsOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiTenantStatisticsAllTenantsSummary)
	return core.NewTryRequest(s.Client, req)
}
