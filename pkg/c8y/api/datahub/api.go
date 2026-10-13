// Package datahub provides access to Cumulocity DataHub: ad-hoc SQL queries,
// Dremio query jobs (the jobs subservice), and the DataHub management API
// (offloading configurations, scheduler jobs, tenant configuration, users).
package datahub

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/datahub/jobs"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

var ApiSQL = "/service/datahub/sql"
var ApiOffloadingConfigurations = "/service/datahub/offloadingconfigurations"
var ApiSchedulerLatestJobs = "/service/datahub/scheduler/latestjobs"
var ApiTenant = "/service/datahub/tenant"
var ApiDremioUsers = "/service/datahub/dremio/users"

const ResultProperty = "rows"

// Service interacts with DataHub.
type Service struct {
	core.Service

	// Jobs manages Dremio query jobs.
	Jobs *jobs.Service
}

// NewService creates a DataHub service.
func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
		Jobs:    jobs.NewService(s),
	}
}

// Query runs a SQL query against the high-performance API and returns the result
// rows. version selects the API version (e.g. "v1"); body carries sql/limit/format.
func (s *Service) Query(ctx context.Context, version string, body any) op.Result[jsonmodels.DataHubResult] {
	return core.ExecuteCollection(ctx, s.queryB(version, body), ResultProperty, "", jsonmodels.NewDataHubResult)
}

func (s *Service) queryB(version string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiSQL)
	if version != "" {
		req.SetQueryParam("version", version)
	}
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// ListOffloadingConfigurationsOptions filters the offloading-configuration
// listing. The endpoint pages with "limit" rather than "pageSize".
type ListOffloadingConfigurationsOptions struct {
	// Last max reported UUID
	LastMaxReportedUUID string `url:"lastMaxReportedUUID,omitempty"`

	// Locale of the returned texts (e.g. "en")
	Locale string `url:"locale,omitempty"`

	// Maximum number of configurations to return
	Limit int `url:"limit,omitempty"`
}

// ListOffloadingConfigurations returns the DataHub offloading configurations
// (the "data.offloadingConfigurations" array of the management API response).
func (s *Service) ListOffloadingConfigurations(ctx context.Context, opt ListOffloadingConfigurationsOptions) op.Result[jsonmodels.DataHubOffloadingConfiguration] {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiOffloadingConfigurations)
	return core.ExecuteCollection(ctx, core.NewTryRequest(s.Client, req, "data.offloadingConfigurations"), "data.offloadingConfigurations", "", jsonmodels.NewDataHubOffloadingConfiguration)
}

// ListLatestJobsOptions filters the scheduler's latest-jobs report.
type ListLatestJobsOptions struct {
	// Job type (e.g. "COMPACTION")
	JobType string `url:"jobType,omitempty"`

	// Offset
	Offset int `url:"offset,omitempty"`

	// Next offset
	NextOffset int `url:"nextOffset,omitempty"`

	// Maximum number of jobs to return
	Limit int `url:"limit,omitempty"`
}

// ListLatestJobs returns the DataHub scheduler's latest job runs. The response
// is a single report document (not a paginated collection) and is returned
// as-is.
func (s *Service) ListLatestJobs(ctx context.Context, opt ListLatestJobsOptions) op.Result[jsonmodels.DataHubSchedulerJobs] {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiSchedulerLatestJobs)
	return core.Execute(ctx, core.NewTryRequest(s.Client, req), jsonmodels.NewDataHubSchedulerJobs)
}

// GetTenant returns the DataHub tenant configuration — the "data" object of
// the management API response envelope ({status, data}), matching the v1 CLI's
// collectionProperty unwrapping. The raw envelope stays available on
// Result.Response.
func (s *Service) GetTenant(ctx context.Context) op.Result[jsonmodels.DataHubTenant] {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenant)
	return core.Execute(ctx, core.NewTryRequest(s.Client, req), func(b []byte) jsonmodels.DataHubTenant {
		if v := gjson.GetBytes(b, "data"); v.Exists() {
			return jsonmodels.NewDataHubTenant([]byte(v.Raw))
		}
		return jsonmodels.NewDataHubTenant(b)
	})
}

// ListUsers returns the DataHub (Dremio) users (the "data.users" array of the
// management API response).
func (s *Service) ListUsers(ctx context.Context) op.Result[jsonmodels.DataHubUser] {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiDremioUsers)
	return core.ExecuteCollection(ctx, core.NewTryRequest(s.Client, req, "data.users"), "data.users", "", jsonmodels.NewDataHubUser)
}
