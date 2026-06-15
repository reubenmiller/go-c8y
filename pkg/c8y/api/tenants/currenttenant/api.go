package currenttenant

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiTenant = "/tenant/currentTenant"

// ApiSystemVersion is the system option holding the platform (backend) version.
var ApiSystemVersion = "/tenant/system/options/system/version"

// ApplicationsResultProperty is the gjson path to the application objects
// embedded in the current-tenant response (applications.references[].application).
const ApplicationsResultProperty = "applications.references.#.application"

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
func (s *Service) Get(ctx context.Context, opt GetOptions) op.Result[jsonmodels.CurrentTenant] {
	return core.Execute(ctx, s.getB(opt), jsonmodels.NewCurrentTenant)
}

func (s *Service) getB(opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTenant)
	return core.NewTryRequest(s.Client, req)
}

// ListApplications lists the applications subscribed to the current tenant. The
// current-tenant endpoint embeds them under applications.references[].application,
// so the collection is extracted from that same response (it is not separately
// paginated).
func (s *Service) ListApplications(ctx context.Context) op.Result[jsonmodels.Application] {
	return core.ExecuteCollection(ctx, s.getB(GetOptions{}), ApplicationsResultProperty, "", jsonmodels.NewApplication)
}

// GetVersion returns the platform (backend) version of the current tenant, which
// the platform exposes as the system option category "system" / key "version".
func (s *Service) GetVersion(ctx context.Context) op.Result[jsonmodels.SystemOption] {
	return core.Execute(ctx, s.getVersionB(), jsonmodels.NewSystemOption)
}

func (s *Service) getVersionB() *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiSystemVersion)
	return core.NewTryRequest(s.Client, req)
}
