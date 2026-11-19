package currentapplication

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// ApiApplicationAuthType is the endpoint for the service user's required
// authentication method.
const ApiApplicationAuthType = "/application/currentApplication/authType"

// SetAuthType sets the authentication method required for the microservice's
// own service user in the current tenant: "NONE" or "CERTIFICATES". It only
// works with service user credentials (not the bootstrap user).
//
// WARNING: "CERTIFICATES" takes effect immediately; the password stops working.
// Obtain and verify a usable certificate for the tenant first. A tenant
// administrator can reset it with applications.Service.ResetAuthType.
func (s *Service) SetAuthType(ctx context.Context, authType string) op.Result[jsonmodels.Application] {
	return httpcore.Execute(ctx, s.setAuthTypeB(authType), jsonmodels.NewApplication)
}

func (s *Service) setAuthTypeB(authType string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(map[string]any{"requiredAuthType": authType}).
		SetURL(ApiApplicationAuthType)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
