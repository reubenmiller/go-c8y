package retentionrules

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiRetentionRules = "/retention/retentions"
var ApiRetentionRule = "/retention/retentions/{id}"

var ParamID = "id"

const ResultProperty = "retentionRules"

func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc: *s,
	}
}

// Service provides api to manage retention rules
type Service struct {
	svc httpcore.Service
}

// ListOptions to filter the retention rules by
type ListOptions struct {
	pagination.PaginationOptions
}

// RetentionRuleIterator provides iteration over retention rules
type RetentionRuleIterator = pagination.Iterator[jsonmodels.RetentionRule]

// Retrieve all login options available in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.RetentionRule] {
	return httpcore.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewRetentionRule)
}

// ListAll returns an iterator for all retention rules
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *RetentionRuleIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.RetentionRule] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewRetentionRule,
	)
}

func (s *Service) listB(opt any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(ApiRetentionRules)
	return httpcore.NewTryRequest(s.svc.Client, req, ResultProperty)
}

// Get a retention rule
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.RetentionRule] {
	return httpcore.Execute(ctx, s.getB(ID), jsonmodels.NewRetentionRule)
}

func (s *Service) getB(ID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, ID).
		SetURL(ApiRetentionRule)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Create a retention rule
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.RetentionRule] {
	return httpcore.Execute(ctx, s.createB(body), jsonmodels.NewRetentionRule)
}

func (s *Service) createB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiRetentionRules)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Update a retention rule
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.RetentionRule] {
	return httpcore.Execute(ctx, s.updateB(ID, body), jsonmodels.NewRetentionRule)
}

func (s *Service) updateB(ID string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, ID).
		SetBody(body).
		SetURL(ApiRetentionRule)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Delete a retention rule
func (s *Service) Delete(ctx context.Context, ID string) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.deleteB(ID)).IgnoreNotFound()
}

func (s *Service) deleteB(ID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamID, ID).
		SetURL(ApiRetentionRule)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
