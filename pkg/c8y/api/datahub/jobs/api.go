// Package jobs provides access to DataHub (Dremio) query jobs.
package jobs

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiSQL = "/service/datahub/dremio/api/v3/sql"
var ApiJob = "/service/datahub/dremio/api/v3/job/{id}"
var ApiJobCancel = "/service/datahub/dremio/api/v3/job/{id}/cancel"
var ApiJobResults = "/service/datahub/dremio/api/v3/job/{id}/results"

const ParamID = "id"

const ResultProperty = "rows"

// Service interacts with DataHub query jobs.
type Service struct {
	svc httpcore.Service
}

// NewService creates a DataHub jobs service.
func NewService(s *httpcore.Service) *Service {
	return &Service{svc: *s}
}

// Create submits a SQL query as a job and returns its id/state.
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.DataHubJob] {
	return httpcore.Execute(ctx, s.createB(body), jsonmodels.NewDataHubJob)
}

func (s *Service) createB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiSQL)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Get returns a job's status by id.
func (s *Service) Get(ctx context.Context, id string) op.Result[jsonmodels.DataHubJob] {
	return httpcore.Execute(ctx, s.getB(id), jsonmodels.NewDataHubJob)
}

func (s *Service) getB(id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamID, id).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiJob)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Cancel cancels a running job.
func (s *Service) Cancel(ctx context.Context, id string) op.Result[jsonmodels.DataHubJob] {
	return httpcore.Execute(ctx, s.cancelB(id), jsonmodels.NewDataHubJob)
}

func (s *Service) cancelB(id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamID, id).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiJobCancel)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// ResultsOptions filters paginated job results.
type ResultsOptions struct {
	Offset int `url:"offset,omitempty"`
}

// GetResults returns the result rows of a completed job.
func (s *Service) GetResults(ctx context.Context, id string, opt ResultsOptions) op.Result[jsonmodels.DataHubResult] {
	return httpcore.ExecuteCollection(ctx, s.getResultsB(id, opt), ResultProperty, "", jsonmodels.NewDataHubResult)
}

func (s *Service) getResultsB(id string, opt ResultsOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamID, id).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiJobResults)
	return httpcore.NewTryRequest(s.svc.Client, req, ResultProperty)
}
