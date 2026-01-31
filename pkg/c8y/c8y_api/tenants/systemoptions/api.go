package systemoptions

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

var ApiSystemOptions = "/tenant/system/options"
var ApiSystemOption = "/tenant/system/options/{category}/{key}"

const ParamKey = "key"
const ParamCategory = "category"

const ResultProperty = "options"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service api to interact with system options
// type Service core.Service
type Service struct {
	core.Service
}

// ListOptions system options filter
type ListOptions struct {
	// Pagination options
	pagination.PaginationOptions
}

// SystemOptionIterator provides iteration over system options
type SystemOptionIterator struct {
	items iter.Seq[jsonmodels.SystemOption]
	err   error
}

func (it *SystemOptionIterator) Items() iter.Seq[jsonmodels.SystemOption] {
	return it.items
}

func (it *SystemOptionIterator) Err() error {
	return it.err
}

func paginateSystemOptions(ctx context.Context, fetch func(page int) op.Result[jsonmodels.SystemOption], maxItems int64) *SystemOptionIterator {
	iterator := &SystemOptionIterator{}

	iterator.items = func(yield func(jsonmodels.SystemOption) bool) {
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
				item := jsonmodels.NewSystemOption(doc.Bytes())
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

// List system options
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.SystemOption] {
	return core.ExecuteReturnCollection(ctx, s.ListB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewSystemOption)
}

// ListAll returns an iterator for all system options
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *SystemOptionIterator {
	if opts.PageSize == 0 {
		opts.PageSize = 2000
	}
	return paginateSystemOptions(ctx, func(page int) op.Result[jsonmodels.SystemOption] {
		opts.CurrentPage = page
		return s.List(ctx, opts)
	}, opts.GetMaxItems())
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiSystemOptions)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type GetOption struct {
	Key      string `url:"-"`
	Category string `url:"-"`
}

// Get a system option
func (s *Service) Get(ctx context.Context, opt GetOption) op.Result[jsonmodels.SystemOption] {
	return core.ExecuteReturnResult(ctx, s.GetB(opt), jsonmodels.NewSystemOption)
}

func (s *Service) GetB(opt GetOption) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamCategory, opt.Category).
		SetPathParam(ParamKey, opt.Key).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiSystemOption)
	return core.NewTryRequest(s.Client, req)
}
