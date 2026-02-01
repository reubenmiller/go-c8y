package devices

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
	"resty.dev/v3"
)

// Obtain a device access token
//
// Only those devices which are registered to use cert auth can authenticate via mTLS protocol and retrieve JWT token.
// Device access token API works only on port 8443 via mutual TLS (mTLS) connection. Immediate issuer of client
// certificate must present in Platform's truststore, if not then whole certificate chain needs to send in header and
// root or any intermediate certificate must be present in the Platform's truststore. We must have the following:
//   - private_key
//   - client certificate
//   - whole certificate chain (Optional - This API requires the client to send a custom header X-SSL-CERT-CHAIN only
//     if the immediate issuer of the client's certificate is not uploaded as a trusted certificate on the platform.
//     If the immediate issuer is already uploaded and trusted, the header can be omitted)
func (s *Service) CreateAccessToken(ctx context.Context) op.Result[jsonmodels.DeviceAccessToken] {
	return core.ExecuteReturnResult(ctx, s.CreateAccessTokenB(), jsonmodels.NewDeviceAccessToken)
}

func (s *Service) CreateAccessTokenB() *core.TryRequest {
	headers := map[string]string{}
	if v, err := certutil.CertificateChain(s.Client.TLSClientConfig().Certificates).Header(); err == nil && len(v) > 0 {
		headers[types.HeaderSSLCertificateChain] = string(v)
	}

	req := s.Service.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeaders(headers).
		SetURL(mtlsEndpoint(s.Client.BaseURL(), ApiDeviceControlAccessToken))
	return core.NewTryRequest(s.Client, req)
}

// mtlsEndpoint returns the host address for the mtls endpoint that can be used for x509 client based authentication
func mtlsEndpoint(fullURL string, paths ...string) string {
	u, err := url.Parse(fullURL)
	if err != nil {
		return fullURL
	}
	out := fmt.Sprintf("%s://%s:%s", u.Scheme, u.Hostname(), "8443")
	if u.Path != "" {
		out = out + "/" + u.Path
	}
	if v, err := url.JoinPath(out, paths...); err == nil {
		return v
	}
	return out
}
