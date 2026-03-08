package core

import (
	"context"
	"io"
	"net/url"

	"github.com/google/go-querystring/query"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/api/contexthelpers"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/realtime"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"resty.dev/v3"
)

type Service struct {
	Client         *resty.Client
	RealtimeClient *realtime.Client
	// MTLSPort is the port used for the mutual-TLS device-certificate token endpoint.
	// Defaults to "8443" when empty.
	MTLSPort string
	// CertChainHeader is the pre-computed value for the X-SSL-CERT-CHAIN header.
	// It is non-empty only when a client certificate with an intermediate chain was
	// supplied via ClientOptions.Auth. Set on the request by any service that
	// requires mTLS (e.g. the device access-token endpoint).
	CertChainHeader string
}

type TryRequest struct {
	Request  *resty.Request
	Client   *resty.Client
	Property string
	// authType overrides the per-request authentication strategy.
	// Zero value (AuthTypeUnset) means "use client default" (token source runs normally).
	authType authentication.AuthType
}

func NewTryRequest(client *resty.Client, req *resty.Request, prop ...string) *TryRequest {
	property := ""
	if len(prop) > 0 {
		property = prop[0]
	}
	return &TryRequest{
		Client:   client,
		Request:  req,
		Property: property,
	}
}

func (r *TryRequest) URL() *url.URL {
	if u, err := url.Parse(r.Request.URL); err == nil {
		return u
	}
	return &url.URL{}
}

func (r *TryRequest) SetContext(ctx context.Context) *TryRequest {
	r.Request.SetContext(ctx)
	// Apply processing mode from context if set
	r.ApplyProcessingModeFromContext(ctx)
	return r
}

// ApplyProcessingModeFromContext sets the processing mode header if one is specified in the context
func (r *TryRequest) ApplyProcessingModeFromContext(ctx context.Context) *TryRequest {
	if mode := ctxhelpers.GetProcessingMode(ctx); mode != "" {
		r.SetProcessingMode(mode)
	}
	return r
}

// WithAuthType forces a specific authentication strategy for this request,
// overriding the client's default token-source behaviour.
//
//   - AuthTypeUnset (default): token source runs normally.
//   - AuthTypeBasic: skip token source; use basic auth (client's credentials or
//     those set by SetBasicAuth). Bearer token is suppressed.
//   - AuthTypeBearer: equivalent to the default — token source runs normally.
//   - AuthTypeNone: no Authorization header is sent (public endpoints).
func (r *TryRequest) WithAuthType(t authentication.AuthType) *TryRequest {
	r.authType = t
	return r
}

// SkipTokenSource is sugar for WithAuthType(AuthTypeBasic). Use it when the
// resty client's configured basic-auth credentials should be applied directly
// without acquiring a bearer token first.
func (r *TryRequest) SkipTokenSource() *TryRequest {
	return r.WithAuthType(authentication.AuthTypeBasic)
}

// WithNoAuth is sugar for WithAuthType(AuthTypeNone). Use it for public
// endpoints that must be reached without any Authorization header.
func (r *TryRequest) WithNoAuth() *TryRequest {
	return r.WithAuthType(authentication.AuthTypeNone)
}

// SetBasicAuth sets explicit basic-auth credentials for this request and marks
// it as AuthTypeBasic so that the token source is bypassed and no bearer token
// overrides the credentials.
func (r *TryRequest) SetBasicAuth(username, password string) *TryRequest {
	// Request-level credentials take precedence over client-level credentials
	// inside resty's addCredentials step.
	r.Request.SetBasicAuth(username, password)
	return r.WithAuthType(authentication.AuthTypeBasic)
}

func (r *TryRequest) SetToken(token string) *TryRequest {
	r.Request.
		SetAuthScheme("Bearer").
		SetAuthToken(token)
	return r
}

func (r *TryRequest) SetProcessingMode(mode types.ProcessingMode) *TryRequest {
	r.Request.SetHeader(types.HeaderProcessingMode, string(mode))
	return r
}

func (r *TryRequest) SetProcessingModePersistent() *TryRequest {
	return r.SetProcessingMode(types.ProcessingModePersistent)
}

func (r *TryRequest) SetProcessingModeTransient() *TryRequest {
	return r.SetProcessingMode(types.ProcessingModeTransient)
}

func (r *TryRequest) SetProcessingModeCEP() *TryRequest {
	return r.SetProcessingMode(types.ProcessingModeCEP)
}

func (r *TryRequest) SetProcessingModeQuiescent() *TryRequest {
	return r.SetProcessingMode(types.ProcessingModeQuiescent)
}

func (r *TryRequest) SetNoResponse() *TryRequest {
	r.Request.Header.Del("Accept")
	return r
}

func (r *TryRequest) SetResponseBodyUnlimitedReads(v bool) *TryRequest {
	r.Request.SetResponseBodyUnlimitedReads(v)
	return r
}

func (r *TryRequest) SetOutputFileName(file string) *TryRequest {
	r.Request.SetOutputFileName(file)
	return r
}

func (r *TryRequest) SetSaveResponse(v bool) *TryRequest {
	r.Request.SetSaveResponse(v)
	return r
}

func (r *TryRequest) Funcs(funcs ...resty.RequestFunc) *TryRequest {
	r.Request.Funcs(funcs...)
	return r
}

func (r *TryRequest) SetResult(v any) *TryRequest {
	r.Request.SetResult(v)
	return r
}

func (r *TryRequest) SetDefaultAcceptHeader() *TryRequest {
	if r.Request.Header.Get("Accept") == "" {
		r.Request.SetHeader("Accept", types.MimeTypeApplicationJSON)
	}
	return r
}

func closeq(v any) {
	if c, ok := v.(io.Closer); ok {
		silently(c.Close())
	}
}

func silently(_ ...any) {}

func (r *TryRequest) Send() (*Response, error) {
	resp, err := r.Request.Send()

	// close body quietly to cleanup results
	closeq(r.Request.Body)

	return &Response{
		Request:  r,
		Response: resp,
	}, err
}

func QueryParameters(opt any) url.Values {
	v, _ := query.Values(opt)
	return v
}
