package currenttenant

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var apiTenant = "/tenant/currentTenant"

// apiSystemVersion is the system option holding the platform (backend) version.
var apiSystemVersion = "/tenant/system/options/system/version"

// ApplicationsResultProperty is the gjson path to the application objects
// embedded in the current-tenant response (applications.references[].application).
const ApplicationsResultProperty = "applications.references.#.application"

func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc: *s,
	}
}

// Service inventory api to interact with the current tenant
// type Service httpcore.Service
type Service struct {
	svc httpcore.Service
}

type GetOptions struct {
	// When set to true, the returned result will contain the parent of the current tenant
	WithParent bool `url:"withParent,omitempty"`
}

// Get current tenant
func (s *Service) Get(ctx context.Context, opt GetOptions) op.Result[jsonmodels.CurrentTenant] {
	return httpcore.Execute(ctx, s.getB(opt), jsonmodels.NewCurrentTenant)
}

func (s *Service) getB(opt GetOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiTenant)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// ListApplications lists the applications subscribed to the current tenant. The
// current-tenant endpoint embeds them under applications.references[].application,
// so the collection is extracted from that same response (it is not separately
// paginated).
func (s *Service) ListApplications(ctx context.Context) op.Result[jsonmodels.Application] {
	return httpcore.ExecuteCollection(ctx, s.getB(GetOptions{}), ApplicationsResultProperty, "", jsonmodels.NewApplication)
}

// GetVersion returns the platform (backend) version of the current tenant, which
// the platform exposes as the system option category "system" / key "version".
func (s *Service) GetVersion(ctx context.Context) op.Result[jsonmodels.SystemOption] {
	return httpcore.Execute(ctx, s.getVersionB(), jsonmodels.NewSystemOption)
}

func (s *Service) getVersionB() *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiSystemVersion)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
