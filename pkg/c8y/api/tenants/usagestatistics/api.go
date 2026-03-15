package usagestatistics

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiTenantStatistics = "/tenant/statistics"
var ApiTenantStatisticsSummary = "/tenant/statistics/summary"
var ApiTenantStatisticsAllTenantsSummary = "/tenant/statistics/allTenantsSummary"

const ParamID = "id"
const ParamChild = "child"

const ResultProperty = "usageStatistics"

// Service
type Service struct{ core.Service }

func NewService(common *core.Service) *Service {
	return &Service{Service: *common}
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
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewTenantUsageStatistics)
}

// ListAll returns an iterator for all tenant usage statistics
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *TenantUsageStatisticsIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.TenantUsageStatistics] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewTenantUsageStatistics,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
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
	return core.Execute(ctx, s.listSummaryB(opt), jsonmodels.NewTenantUsageStatistics)
}

func (s *Service) listSummaryB(opt ListSummaryOptions) *core.TryRequest {
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
	return core.ExecuteCollection(ctx, s.listSummaryAllTenantsB(opt), "", "", jsonmodels.NewTenantUsageStatisticsSummary)
}

func (s *Service) listSummaryAllTenantsB(opt ListSummaryAllTenantsOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiTenantStatisticsAllTenantsSummary)
	return core.NewTryRequest(s.Client, req)
}
