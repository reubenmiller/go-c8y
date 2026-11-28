package systemoptions

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var apiSystemOptions = "/tenant/system/options"
var apiSystemOption = "/tenant/system/options/{category}/{key}"

const paramKey = "key"
const paramCategory = "category"

const resultProperty = "options"

func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc: *s,
	}
}

// Service api to interact with system options
// type Service httpcore.Service
type Service struct {
	svc httpcore.Service
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
	return httpcore.ExecuteCollection(ctx, s.listB(opt), resultProperty, types.ResponseFieldStatistics, jsonmodels.NewSystemOption)
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

func (s *Service) listB(opt ListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiSystemOptions)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

// GetOptions are the options of [Service.Get].
type GetOptions struct {
	Key      string `url:"-"`
	Category string `url:"-"`
}

// Get a system option
func (s *Service) Get(ctx context.Context, opt GetOptions) op.Result[jsonmodels.SystemOption] {
	return httpcore.Execute(ctx, s.getB(opt), jsonmodels.NewSystemOption)
}

func (s *Service) getB(opt GetOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(paramCategory, opt.Category).
		SetPathParam(paramKey, opt.Key).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiSystemOption)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
