package currenttenant

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiTenant = "/tenant/currentTenant"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service inventory api to interact with the current tenant
// type Service core.Service
type Service struct {
	core.Service
}

type GetOptions struct {
	// When set to true, the returned result will contain the parent of the current tenant
	WithParent bool `url:"withParent,omitempty"`
}

// Get current tenant
func (s *Service) Get(ctx context.Context, opt GetOptions) op.Result[jsonmodels.Tenant] {
	return core.Execute(ctx, s.getB(opt), jsonmodels.NewTenant)
}

func (s *Service) getB(opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenant)
	return core.NewTryRequest(s.Client, req)
}
