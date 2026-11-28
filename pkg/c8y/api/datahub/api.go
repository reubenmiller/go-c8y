// Package datahub provides access to Cumulocity DataHub: ad-hoc SQL queries,
// Dremio query jobs (the jobs subservice), and the DataHub management API
// (offloading configurations, scheduler jobs, tenant configuration, users).
package datahub

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/datahub/jobs"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

var apiSQL = "/service/datahub/sql"
var apiOffloadingConfigurations = "/service/datahub/offloadingconfigurations"
var apiSchedulerLatestJobs = "/service/datahub/scheduler/latestjobs"
var apiTenant = "/service/datahub/tenant"
var apiDremioUsers = "/service/datahub/dremio/users"

const resultProperty = "rows"

// Service interacts with DataHub.
type Service struct {
	svc httpcore.Service
	// Jobs manages Dremio query jobs.
	Jobs *jobs.Service
}

// NewService creates a DataHub service.
func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc:  *s,
		Jobs: jobs.NewService(s),
	}
}

// Query runs a SQL query against the high-performance API and returns the result
// rows. version selects the API version (e.g. "v1"); body carries sql/limit/format.
func (s *Service) Query(ctx context.Context, version string, body any) op.Result[jsonmodels.DataHubResult] {
	return httpcore.ExecuteCollection(ctx, s.queryB(version, body), resultProperty, "", jsonmodels.NewDataHubResult)
}

func (s *Service) queryB(version string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiSQL)
	if version != "" {
		req.SetQueryParam("version", version)
	}
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
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
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiOffloadingConfigurations)
	return httpcore.ExecuteCollection(ctx, httpcore.NewTryRequest(s.svc.Client, req, "data.offloadingConfigurations"), "data.offloadingConfigurations", "", jsonmodels.NewDataHubOffloadingConfiguration)
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
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiSchedulerLatestJobs)
	return httpcore.Execute(ctx, httpcore.NewTryRequest(s.svc.Client, req), jsonmodels.NewDataHubSchedulerJobs)
}

// GetTenant returns the DataHub tenant configuration — the "data" object of
// the management API response envelope ({status, data}), matching the v1 CLI's
// collectionProperty unwrapping. The raw envelope stays available on
// Result.Response.
func (s *Service) GetTenant(ctx context.Context) op.Result[jsonmodels.DataHubTenant] {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiTenant)
	return httpcore.Execute(ctx, httpcore.NewTryRequest(s.svc.Client, req), func(b []byte) jsonmodels.DataHubTenant {
		if v := gjson.GetBytes(b, "data"); v.Exists() {
			return jsonmodels.NewDataHubTenant([]byte(v.Raw))
		}
		return jsonmodels.NewDataHubTenant(b)
	})
}

// ListUsers returns the DataHub (Dremio) users (the "data.users" array of the
// management API response).
func (s *Service) ListUsers(ctx context.Context) op.Result[jsonmodels.DataHubUser] {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiDremioUsers)
	return httpcore.ExecuteCollection(ctx, httpcore.NewTryRequest(s.svc.Client, req, "data.users"), "data.users", "", jsonmodels.NewDataHubUser)
}
