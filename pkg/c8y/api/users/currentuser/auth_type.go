package currentuser

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// Values of requiredAuthType.
const (
	AuthTypeNone         = "NONE"
	AuthTypeCertificates = "CERTIFICATES"
)

// SetRequiredAuthType sets the authentication method required for the current
// user: AuthTypeNone or AuthTypeCertificates.
//
// WARNING: AuthTypeCertificates takes effect immediately. From then on only
// tokens from a certificate exchange are accepted; password-derived tokens are
// rejected. If the certificate is lost, the "forgot password" email flow
// resets the value to NONE. Prefer api.Client.EnableCertificateAuth, which
// checks the certificate first.
func (s *Service) SetRequiredAuthType(ctx context.Context, authType string) op.Result[jsonmodels.User] {
	return s.Update(ctx, map[string]any{"requiredAuthType": authType})
}
