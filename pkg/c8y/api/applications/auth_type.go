package applications

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// ApiApplicationAuthType is the endpoint to reset a service user's required
// authentication method.
const ApiApplicationAuthType = "/application/applications/{id}/authType"

// ResetAuthType resets the authentication method of the service user of the
// given application (in the current tenant) to "NONE". This is the recovery
// path for a service user that switched to certificates and lost its
// certificate. Setting "CERTIFICATES" here is not allowed by the platform.
func (s *Service) ResetAuthType(ctx context.Context, id string) op.Result[jsonmodels.Application] {
	return httpcore.Execute(ctx, s.resetAuthTypeB(id), jsonmodels.NewApplication)
}

func (s *Service) resetAuthTypeB(id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam("id", id).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(map[string]any{"requiredAuthType": "NONE"}).
		SetURL(ApiApplicationAuthType)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
