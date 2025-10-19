package logintokens

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
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
}

// Obtain an OAI-Secure access token
func (s *Service) Create(ctx context.Context, opt CreateTokenOptions) (*model.OAIToken, error) {
	return core.ExecuteResultOnly[model.OAIToken](ctx, s.CreateB(opt))
}

// Obtain an OAI-Secure access token
func (s *Service) CreateB(opt CreateTokenOptions) *core.TryRequest {
	params := map[string]string{}
	if opt.Tenant != "" {
		params["tenant_id"] = opt.Tenant
	}
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetQueryParams(params).
		SetFormDataFromValues(core.QueryParameters(opt)).
		Funcs(core.NoAuthorization()).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiOAuthToken)
	// The response might not have the Content-Type set
	req.ForceResponseContentType = types.MimeTypeApplicationJSON
	return core.NewTryRequest(s.Client, req)
}

var (
	CookieAuthorization = "authorization"
	CookieXSRFToken     = "XSRF-TOKEN"
)

// Obtain an OAI-Secure and XSRF tokens in cookies
func (s *Service) CreateCookies(ctx context.Context, opt CreateTokenOptions) error {
	resp, err := core.ExecuteResponseOnly(ctx, s.CreateCookiesB(opt))
	if err != nil {
		return err
	}
	s.Client.SetCookies(resp.Cookies())
	return nil
}

// Obtain an OAI-Secure and XSRF tokens in cookies
func (s *Service) CreateCookiesB(opt CreateTokenOptions) *core.TryRequest {
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
	return core.NewTryRequest(s.Client, req)
}
