package logintokens

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// apiOAuthCertificate is the certificate token exchange endpoint.
var apiOAuthCertificate = "/tenant/oauth/certificate"

// HeaderTFACode carries the second factor for the certificate exchange.
const HeaderTFACode = "X-Cumulocity-TFA-Code"

// CertificateTokenOptions are the inputs of the certificate token exchange.
type CertificateTokenOptions struct {
	// Tenant is the tenant ID the account belongs to (required).
	Tenant string

	// Chain is the leaf certificate, or the full chain, in PEM format.
	Chain []byte

	// TFACode is the current TOTP or SMS code, when TFA is active for the user.
	TFACode string
}

// CreateWithCertificate exchanges an X.509 certificate for an access token over
// standard HTTPS. The account must have opted in to certificate authentication
// (requiredAuthType CERTIFICATES), except device users.
//
// The returned token is a JWE encrypted to the certificate's RSA public key;
// decrypt it with the private key. api.Client does this when logging in with
// authentication.LoginMethodCertificate.
func (s *Service) CreateWithCertificate(ctx context.Context, opt CertificateTokenOptions) op.Result[jsonmodels.CertificateAccessToken] {
	return httpcore.Execute(ctx, s.createWithCertificateB(opt), jsonmodels.NewCertificateAccessToken)
}

func (s *Service) createWithCertificateB(opt CertificateTokenOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetQueryParam("tenant_id", opt.Tenant).
		SetHeader("Content-Type", types.MimeTypePEMFile).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(opt.Chain).
		SetURL(apiOAuthCertificate)
	if opt.TFACode != "" {
		req.SetHeader(HeaderTFACode, opt.TFACode)
	}
	req.SetResponseForceContentType(types.MimeTypeApplicationJSON)
	return httpcore.NewTryRequest(s.svc.Client, req).WithNoAuth()
}
