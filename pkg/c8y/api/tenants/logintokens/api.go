package logintokens

import (
	"context"
	"net/http"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiOAuth = "/tenant/oauth"
var ApiOAuthToken = "/tenant/oauth/token"

var ParamId = "id"

const ResultProperty = "events"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service provides api to get/set/delete events in Cumulocity
type Service struct {
	core.Service
}

type CreateTokenOptions struct {
	// Unique identifier of a Cumulocity tenant. If not provided, the tenant is calculated based on the request domain
	Tenant string `url:"-"`

	// Used in case of SSO login. A code received from the external authentication server is exchanged to an internal access token
	Code string `url:"code,omitempty"`

	// Dependent on the authentication type. PASSWORD is used for OAI-Secure
	GrantType string `url:"grant_type,omitempty"`

	// Used in case of OAI-Secure authentication
	Password string `url:"password,omitempty"`

	// Current TFA code, sent by the user, if a TFA code is required to log in. Used in case of OAI-Secure authentication
	TFACode string `url:"tfa_code,omitempty"`

	// Used in case of OAI-Secure authentication
	Username string `url:"username,omitempty"`

	// Value to be used in the REQUEST_ORIGIN cookie to indicate the request origin
	// Typically used during OAuth2
	RequestOrigin string `url:"-"`

	// CodeVerifier is the PKCE code verifier (RFC 7636). Set when the
	// authorization request was made with a code_challenge; the server uses
	// this to verify the exchange. Leave empty when PKCE is not in use.
	CodeVerifier string `url:"code_verifier,omitempty"`
}

// Obtain an OAI-Secure access token
func (s *Service) Create(ctx context.Context, opt CreateTokenOptions) op.Result[jsonmodels.OAIToken] {
	return core.Execute(ctx, s.createB(opt), jsonmodels.NewOAIToken)
}

// Obtain an OAI-Secure access token
func (s *Service) createB(opt CreateTokenOptions) *core.TryRequest {
	params := map[string]string{}
	if opt.Tenant != "" {
		params["tenant_id"] = opt.Tenant
	}
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetQueryParams(params).
		SetFormDataFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiOAuthToken).
		SetRetryCount(1).
		SetRetryDelayStrategy(func(r *resty.Response, err error) (time.Duration, error) {
			return 100 * time.Millisecond, nil
		}).
		SetAllowNonIdempotentRetry(true).
		AddRetryConditions(func(r *resty.Response, err error) bool {
			// FIXME: retry as sometimes the server's first response to the authorization provider
			// can fail, but subsequent requests are ok
			shouldRetry := r != nil && r.StatusCode() == 400
			return shouldRetry
		})

	if opt.RequestOrigin != "" {
		req.SetCookie(&http.Cookie{
			Name:  "REQUEST_ORIGIN",
			Value: opt.RequestOrigin,
		})
	}

	// The response might not have the Content-Type set
	req.ForceResponseContentType = types.MimeTypeApplicationJSON
	return core.NewTryRequest(s.Client, req).WithNoAuth()
}

var (
	CookieAuthorization = "authorization"
	CookieXSRFToken     = "XSRF-TOKEN"
)

// Obtain an OAI-Secure and XSRF tokens in cookies
func (s *Service) CreateCookies(ctx context.Context, opt CreateTokenOptions) error {
	resp, err := core.ExecuteResponseOnly(ctx, s.createCookiesB(opt))
	if err != nil {
		return err
	}
	s.Client.SetCookies(resp.Cookies())
	return nil
}

// Obtain an OAI-Secure and XSRF tokens in cookies
func (s *Service) createCookiesB(opt CreateTokenOptions) *core.TryRequest {
	params := map[string]string{}
	if opt.Tenant != "" {
		params["tenant_id"] = opt.Tenant
	}
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetQueryParams(params).
		SetFormDataFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiOAuth)
	return core.NewTryRequest(s.Client, req).WithNoAuth()
}
