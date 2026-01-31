package systemoptions

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
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
type ListOptions struct{}

// List system options
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.SystemOption] {
	return core.ExecuteReturnCollection(ctx, s.ListB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewSystemOption)
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
