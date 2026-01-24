package usagestatistics

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
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

// List tenant statistics
func (s *Service) List(ctx context.Context, opt ListOptions) (*model.TenantUsageStatisticsCollection, error) {
	return core.ExecuteResultOnly[model.TenantUsageStatisticsCollection](ctx, s.ListB(opt))
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
func (s *Service) ListSummary(ctx context.Context, opt ListSummaryOptions) (*model.TenantUsageStatistics, error) {
	return core.ExecuteResultOnly[model.TenantUsageStatistics](ctx, s.ListSummaryB(opt))
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
func (s *Service) ListSummaryAllTenants(ctx context.Context, opt ListSummaryAllTenantsOptions) ([]model.TenantUsageStatisticsSummary, error) {
	return core.ExecuteResultsArrayOnly[model.TenantUsageStatisticsSummary](ctx, s.ListSummaryAllTenantsB(opt))
}

func (s *Service) ListSummaryAllTenantsB(opt ListSummaryAllTenantsOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiTenantStatisticsAllTenantsSummary)
	return core.NewTryRequest(s.Client, req)
}
