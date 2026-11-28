package usagestatistics

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var apiTenantStatistics = "/tenant/statistics"
var apiTenantStatisticsSummary = "/tenant/statistics/summary"
var apiTenantStatisticsAllTenantsSummary = "/tenant/statistics/allTenantsSummary"

const paramID = "id"
const paramChild = "child"

const resultProperty = "usageStatistics"

// Service
type Service struct{ svc httpcore.Service }

func NewService(common *httpcore.Service) *Service {
	return &Service{svc: *common}
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
	return httpcore.ExecuteCollection(ctx, s.listB(opt), resultProperty, types.ResponseFieldStatistics, jsonmodels.NewTenantUsageStatistics)
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

func (s *Service) listB(opt ListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiTenantStatistics)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

// ListSummaryOptions
type ListSummaryOptions struct {
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	Tenant string `url:"tenant,omitempty,omitzero"`
}

// List tenant statistics
func (s *Service) ListSummary(ctx context.Context, opt ListSummaryOptions) op.Result[jsonmodels.TenantUsageStatistics] {
	return httpcore.Execute(ctx, s.listSummaryB(opt), jsonmodels.NewTenantUsageStatistics)
}

func (s *Service) listSummaryB(opt ListSummaryOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiTenantStatisticsSummary)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

// ListSummaryAllTenantsOptions
type ListSummaryAllTenantsOptions struct {
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	DateTo time.Time `url:"dateTo,omitempty,omitzero"`
}

// // List usage statistics of all tenants
func (s *Service) ListSummaryAllTenants(ctx context.Context, opt ListSummaryAllTenantsOptions) op.Result[jsonmodels.TenantUsageStatisticsSummary] {
	return httpcore.ExecuteCollection(ctx, s.listSummaryAllTenantsB(opt), "", "", jsonmodels.NewTenantUsageStatisticsSummary)
}

func (s *Service) listSummaryAllTenantsB(opt ListSummaryAllTenantsOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiTenantStatisticsAllTenantsSummary)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
