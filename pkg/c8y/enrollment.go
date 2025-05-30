package c8y

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/certutil"
	"github.com/tidwall/gjson"
	"go.mozilla.org/pkcs7"
)

// DeviceEnrollmentService provides enrollment function to enroll new devices and receive a device certificate
type DeviceEnrollmentService service

// IdentityOptions Identity parameters required when creating a new external id
type EnrollmentOptions struct {
	ExternalID string `json:"externalId"`
	Type       string `json:"type"`
}

// Identity Cumulocity Identity object holding the information about the external id and link to the managed object
type Enrollment struct {
	ExternalID    string            `json:"externalId"`
	Type          string            `json:"type"`
	Self          string            `json:"self"`
	ManagedObject IdentityReference `json:"managedObject"`

	Item gjson.Result `json:"-"`
}

// Create adds a new external id for the given managed object id
func (s *DeviceEnrollmentService) Enroll(ctx context.Context, externalID string, oneTimePassword string, csr *x509.CertificateRequest) (*x509.Certificate, *Response, error) {
	headers := http.Header{}
	headers.Add("Content-Transfer-Encoding", "base64")
	reqContext := NewBasicAuthAuthorizationContext(ctx, "", externalID, oneTimePassword)

	resp, err := s.client.SendRequest(reqContext, RequestOptions{
		Method:      "POST",
		Path:        ".well-known/est/simpleenroll",
		Header:      headers,
		ContentType: "application/pkcs10",
		Body:        base64.StdEncoding.EncodeToString(csr.Raw),
	})

	if err != nil {
		return nil, resp, err
	}

	return s.parsePKCS7Response(resp)
}

// Re enrollment options
type ReEnrollOptions struct {
	// Token to use for authorization
	Token string

	// Certificate Signing Request to request a new certificate
	CSR *x509.CertificateRequest
}

// ReEnroll an already enrolled device using an existing device certificate
// If the token is left empty, then the current user's credentials will be used, however the request will fail if the user does
// not have the following role: ROLE_DEVICE
func (s *DeviceEnrollmentService) ReEnroll(ctx context.Context, opts ReEnrollOptions) (*x509.Certificate, *Response, error) {
	if opts.CSR == nil {
		return nil, nil, fmt.Errorf("no certificate signing request was provided")
	}

	var reqContext context.Context
	if opts.Token != "" {
		reqContext = NewBearerAuthAuthorizationContext(ctx, opts.Token)
	} else {

		reqContext = ctx
	}

	headers := http.Header{}
	headers.Add("Content-Transfer-Encoding", "base64")

	resp, err := s.client.SendRequest(reqContext, RequestOptions{
		Method:      "POST",
		Path:        ".well-known/est/simplereenroll",
		Header:      headers,
		ContentType: "application/pkcs10",
		Body:        base64.StdEncoding.EncodeToString(opts.CSR.Raw),
	})

	if err != nil {
		return nil, resp, err
	}

	return s.parsePKCS7Response(resp)
}

// AccessToken device access token
type AccessToken struct {
	AccessToken string `json:"accessToken,omitempty"`
}

// RequestAccessToken using an x509 client certificate
// If the clientCert is to nil, then the current client will be used.
//
// If the uploaded trusted certificate is not an immediate issuer of the device
// certificate but belongs to the device’s chain of trust, then the device must
// send the entire certificate chain in the 'X-Ssl-Cert-Chain' to be authenticated
// successfully and retrieve the device access token via the headers argument
//
// See https://cumulocity.com/docs/device-integration/device-integration-rest/#device-authentication for more details
func (s *DeviceEnrollmentService) RequestAccessToken(ctx context.Context, clientCert *tls.Certificate, headers *http.Header) (*AccessToken, *Response, error) {
	deviceClient := s.client
	if clientCert != nil {
		// Create a new client which uses the given certificate
		// Use similar setting as the main client for consistency
		skipVerify := false
		if s.client.client.Transport.(*http.Transport).TLSClientConfig != nil {
			skipVerify = s.client.client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify
		}

		httpClient := NewHTTPClient(
			WithClientCertificate(*clientCert),
			WithInsecureSkipVerify(skipVerify),
		)
		deviceClient = NewClientFromOptions(httpClient, ClientOptions{
			BaseURL: s.client.BaseURL.String(),
		})
	}

	if headers == nil {
		headers = &http.Header{}
	}

	data := new(AccessToken)
	resp, err := deviceClient.SendRequest(context.Background(), RequestOptions{
		Method:       http.MethodPost,
		Path:         "devicecontrol/deviceAccessToken",
		Host:         mtlsEndpoint(s.client.BaseURL),
		Header:       *headers,
		ResponseData: data,

		// No auth is required as x509 certificates are being used
		NoAuthentication: true,
	})
	return data, resp, err
}

// mtlsEndpoint returns the host address for the mtls endpoint that can be used for x509 client based authentication
func mtlsEndpoint(u *url.URL) string {
	out := fmt.Sprintf("%s://%s:%s", u.Scheme, u.Hostname(), "8443")
	if u.Path != "" {
		out = out + "/" + u.Path
	}
	return out
}

func (s *DeviceEnrollmentService) parsePKCS7Response(resp *Response) (*x509.Certificate, *Response, error) {
	var err error
	// Decode response
	var contents []byte

	if transferEncoding := resp.Response.Header.Get("Content-Transfer-Encoding"); transferEncoding == "base64" {
		v, decodeErr := certutil.Base64Decode(resp.Body())
		if decodeErr != nil {
			return nil, resp, fmt.Errorf("failed to decode response using base64. %w", decodeErr)
		}
		contents = v
	} else {
		contents = resp.Body()
	}

	// Parse certificate
	var cert *x509.Certificate
	contentType := resp.Response.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "application/pkcs7-mime") {
		p7, p7Err := pkcs7.Parse(contents)
		if p7Err != nil {
			return nil, resp, p7Err
		}

		if len(p7.Certificates) == 0 {
			return nil, resp, fmt.Errorf("response did not contain any x509 certificates")
		}

		cert = p7.Certificates[0]
	} else if strings.HasPrefix(contentType, "application/pkcs10") {
		cert, err = certutil.ParseCertificatePEM(contents)
	}

	return cert, resp, err
}
