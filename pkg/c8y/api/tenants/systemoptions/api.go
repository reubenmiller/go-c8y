package systemoptions

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
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
type SystemOptionIterator = pagination.Iterator[jsonmodels.SystemOption]

// List system options
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.SystemOption] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewSystemOption)
}

// ListAll returns an iterator for all system options
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *SystemOptionIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.SystemOption] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewSystemOption,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
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
	return core.Execute(ctx, s.getB(opt), jsonmodels.NewSystemOption)
}

func (s *Service) getB(opt GetOption) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamCategory, opt.Category).
		SetPathParam(ParamKey, opt.Key).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiSystemOption)
	return core.NewTryRequest(s.Client, req)
}
