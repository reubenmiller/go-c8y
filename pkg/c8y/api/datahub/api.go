// Package datahub provides access to Cumulocity DataHub: ad-hoc SQL queries and
// Dremio query jobs (the jobs subservice).
package datahub

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/datahub/jobs"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiSQL = "/service/datahub/sql"

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
