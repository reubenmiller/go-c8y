package retentionrules

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiRetentionRules = "/retention/retentions"
var ApiRetentionRule = "/retention/retentions/{id}"

var ParamId = "id"

const ResultProperty = "retentionRules"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service provides api to manage retention rules
type Service struct {
	core.Service
}

// ListOptions to filter the retention rules by
type ListOptions struct {
	pagination.PaginationOptions
}

// RetentionRuleIterator provides iteration over retention rules
type RetentionRuleIterator = pagination.Iterator[jsonmodels.RetentionRule]

// Retrieve all login options available in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.RetentionRule] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewRetentionRule)
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

func (s *Service) listB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiRetentionRules)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get a retention rule
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.RetentionRule] {
	return core.Execute(ctx, s.getB(ID), jsonmodels.NewRetentionRule)
}

func (s *Service) getB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, ID).
		SetURL(ApiRetentionRule)
	return core.NewTryRequest(s.Client, req)
}

// Create a retention rule
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.RetentionRule] {
	return core.Execute(ctx, s.createB(body), jsonmodels.NewRetentionRule)
}

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiRetentionRules)
	return core.NewTryRequest(s.Client, req)
}

// Update a retention rule
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.RetentionRule] {
	return core.Execute(ctx, s.updateB(ID, body), jsonmodels.NewRetentionRule)
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, ID).
		SetBody(body).
		SetURL(ApiRetentionRule)
	return core.NewTryRequest(s.Client, req)
}

// Delete a retention rule
func (s *Service) Delete(ctx context.Context, ID string) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(ID))
}

func (s *Service) deleteB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, ID).
		SetURL(ApiRetentionRule)
	return core.NewTryRequest(s.Client, req)
}
