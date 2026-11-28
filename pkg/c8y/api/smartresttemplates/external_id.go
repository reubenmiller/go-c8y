package smartresttemplates

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// apiTemplateExternalID binds a SmartREST 1.0 template X-Id to a managed object.
var apiTemplateExternalID = "/inventory/templates/globalId/{globalId}/externalId"

// ExternalIDTypeSmartRest is the external ID type of SmartREST template X-Ids.
const ExternalIDTypeSmartRest = "c8y_SmartRestDeviceIdentifier"

// BindExternalID binds a SmartREST 1.0 template identifier (X-Id) to the
// existing managed object globalID.
func (s *Service) BindExternalID(ctx context.Context, globalID, xID string) op.Result[jsonmodels.Identity] {
	return httpcore.Execute(ctx, s.bindExternalIDB(globalID, xID), jsonmodels.NewIdentity)
}

func (s *Service) bindExternalIDB(globalID, xID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Content-Type", "application/vnd.com.nsn.cumulocity.externalid+json").
		SetHeader("Accept", "application/json").
		SetBody(map[string]string{"externalId": xID, "type": ExternalIDTypeSmartRest}).
		SetPathParam("globalId", globalID).
		SetURL(apiTemplateExternalID)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
