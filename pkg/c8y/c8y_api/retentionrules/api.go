package retentionrules

import (
	"context"
	"iter"
	"log/slog"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
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
type RetentionRuleIterator struct {
	items iter.Seq[jsonmodels.RetentionRule]
	err   error
}

func (it *RetentionRuleIterator) Items() iter.Seq[jsonmodels.RetentionRule] {
	return it.items
}

func (it *RetentionRuleIterator) Err() error {
	return it.err
}

func paginateRetentionRules(ctx context.Context, fetch func(page int) op.Result[jsonmodels.RetentionRule], maxItems int64) *RetentionRuleIterator {
	iterator := &RetentionRuleIterator{}

	iterator.items = func(yield func(jsonmodels.RetentionRule) bool) {
		page := 1
		count := int64(0)
		for {
			result := fetch(page)
			if result.Err != nil {
				iterator.err = result.Err
				return
			}
			countBeforeResults := count
			for doc := range result.Data.Iter() {
				if maxItems > 0 && count >= maxItems {
					return
				}
				item := jsonmodels.NewRetentionRule(doc.Bytes())
				if !yield(item) {
					return
				}
				count++
			}
			if countBeforeResults == count {
				slog.Info("Stopping pagination as results array is empty")
				return
			}

			totalPages, ok := result.Meta["totalPages"].(int64)
			if ok && page >= int(totalPages) {
				return
			}
			page++
		}
	}

	return iterator
}

// Retrieve all login options available in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.RetentionRule] {
	return core.ExecuteReturnCollection(ctx, s.ListB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewRetentionRule)
}

// ListAll returns an iterator for all retention rules
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *RetentionRuleIterator {
	if opts.PageSize == 0 {
		opts.PageSize = 2000
	}
	return paginateRetentionRules(ctx, func(page int) op.Result[jsonmodels.RetentionRule] {
		opts.CurrentPage = page
		return s.List(ctx, opts)
	}, opts.GetMaxItems())
}

func (s *Service) ListB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiRetentionRules)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get a retention rule
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.RetentionRule] {
	return core.ExecuteReturnResult(ctx, s.GetB(ID), jsonmodels.NewRetentionRule)
}

func (s *Service) GetB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiRetentionRule)
	return core.NewTryRequest(s.Client, req)
}

// Create a retention rule
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.RetentionRule] {
	return core.ExecuteReturnResult(ctx, s.CreateB(body), jsonmodels.NewRetentionRule)
}

func (s *Service) CreateB(body any) *core.TryRequest {
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
	return core.ExecuteReturnResult(ctx, s.UpdateB(ID, body), jsonmodels.NewRetentionRule)
}

func (s *Service) UpdateB(ID string, body any) *core.TryRequest {
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
func (s *Service) Delete(ctx context.Context, ID string) op.Result[jsonmodels.RetentionRule] {
	return core.ExecuteReturnResult(ctx, s.DeleteB(ID), jsonmodels.NewRetentionRule)
}

func (s *Service) DeleteB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, ID).
		SetURL(ApiRetentionRule)
	return core.NewTryRequest(s.Client, req)
}
