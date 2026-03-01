// Package devicestatistics provides a client service for the Cumulocity
// Device Statistics API.
//
// The package covers two endpoints:
//
//  1. Monthly device statistics per tenant:
//     GET /tenant/statistics/device/{tenantId}/monthly/{date}
//
//  2. Daily device statistics per tenant:
//     GET /tenant/statistics/device/{tenantId}/daily/{date}
//
// Both endpoints return a DeviceStatisticsCollection which is optionally
// filterable by a specific device ID.
//
// Required roles: ROLE_TENANT_STATISTICS_READ
package devicestatistics

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

// Endpoint templates — {tenantId} and {date} are path parameters.
var (
	ApiDeviceStatisticsMonthly = "/tenant/statistics/device/{tenantId}/monthly/{date}"
	ApiDeviceStatisticsDaily   = "/tenant/statistics/device/{tenantId}/daily/{date}"
)

// Path-parameter names.
const (
	ParamTenantID = "tenantId"
	ParamDate     = "date"
)

// ResultProperty is the JSON key wrapping the device-statistics array in a
// collection response.
//
// OAS DeviceStatisticsCollection uses "statistics" as the array key (the same
// name as the paging-statistics object used in other collection types, but
// DeviceStatisticsCollection has no separate paging-statistics object).
const ResultProperty = "statistics"

// NewService creates a new Service backed by the provided core.Service.
func NewService(s *core.Service) *Service {
	return &Service{Service: *s}
}

// Service provides access to the Cumulocity Device Statistics API.
type Service struct {
	core.Service
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// DeviceStatisticsIterator is a lazy iterator over a (potentially multi-page)
// collection of device statistics.
type DeviceStatisticsIterator = pagination.Iterator[jsonmodels.DeviceStatistics]

// ListOptions carries the common query parameters for both monthly and daily
// device-statistics endpoints.
type ListOptions struct {
	// TenantID is the ID of the tenant to query. Leave empty to use the
	// client default (the tenant from the configured credentials).
	TenantID string `url:"-"`

	// Date is the date of the query in YYYY-MM-dd format.
	// For monthly queries the day component is ignored.
	Date string `url:"-"`

	// DeviceID optionally filters the results to a single device.
	DeviceID string `url:"deviceId,omitempty"`

	// Pagination options (currentPage, pageSize, withTotalPages).
	pagination.PaginationOptions
}

// ---------------------------------------------------------------------------
// Monthly
// ---------------------------------------------------------------------------

// ListMonthly retrieves a single page of device statistics for the given
// month. The Date field of opt must be in YYYY-MM-dd format; the day
// component is ignored by the platform.
func (s *Service) ListMonthly(ctx context.Context, opt ListOptions) op.Result[jsonmodels.DeviceStatistics] {
	return core.ExecuteCollection(ctx, s.listMonthlyB(opt), ResultProperty, "", jsonmodels.NewDeviceStatistics)
}

// ListAllMonthly returns a lazy iterator that transparently pages through all
// monthly device statistics for the given month.
func (s *Service) ListAllMonthly(ctx context.Context, opts ListOptions) *DeviceStatisticsIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.DeviceStatistics] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.ListMonthly(ctx, o)
		},
		jsonmodels.NewDeviceStatistics,
	)
}

func (s *Service) listMonthlyB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantID, opt.TenantID).
		SetPathParam(ParamDate, opt.Date).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiDeviceStatisticsMonthly)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// ---------------------------------------------------------------------------
// Daily
// ---------------------------------------------------------------------------

// ListDaily retrieves a single page of device statistics for the given day.
// The Date field of opt must be in YYYY-MM-dd format.
func (s *Service) ListDaily(ctx context.Context, opt ListOptions) op.Result[jsonmodels.DeviceStatistics] {
	return core.ExecuteCollection(ctx, s.listDailyB(opt), ResultProperty, "", jsonmodels.NewDeviceStatistics)
}

// ListAllDaily returns a lazy iterator that transparently pages through all
// daily device statistics for the given day.
func (s *Service) ListAllDaily(ctx context.Context, opts ListOptions) *DeviceStatisticsIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.DeviceStatistics] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.ListDaily(ctx, o)
		},
		jsonmodels.NewDeviceStatistics,
	)
}

func (s *Service) listDailyB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTenantID, opt.TenantID).
		SetPathParam(ParamDate, opt.Date).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiDeviceStatisticsDaily)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}
