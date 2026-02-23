package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/loginoptions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/pkg/oauth/api"
	oauth2_api "github.com/reubenmiller/go-c8y/pkg/oauth/api"
	"github.com/reubenmiller/go-c8y/pkg/oauth/device"
	"github.com/tidwall/gjson"
)

var ErrSSOInvalidConfiguration = errors.New("invalid sso configuration")

// GetLoginOptions returns the login options available for the tenant
func getAuthorizationRequest(ctx context.Context, client *http.Client, oauthUrl string) (*api.AuthorizationRequest, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", oauthUrl, nil)
	if err != nil {
		return nil, err
	}

	if client == nil {
		client = http.DefaultClient
	}

	// Disable redirects so we can capture the first redirect location
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location, err := resp.Location()
		if err != nil {
			return nil, fmt.Errorf("failed to get redirect location: %w", err)
		}
		return getAuthorizationEndpointFromURL(location), nil
	} else if resp.StatusCode == 200 {
		// Read redirectTo from the response body
		if b, err := io.ReadAll(resp.Body); err == nil {
			if value := gjson.GetBytes(b, "redirectTo"); value.Exists() {
				u, err := url.Parse(value.String())
				if err == nil {

					// remove the redirect_uri if found.
					params := u.Query()
					// TODO: Check if the redirect uri is needed here
					// params.Set("redirect_uri", redirectURI)
					u.RawQuery = params.Encode()

					return getAuthorizationEndpointFromURL(u), nil
				}
			}
		}
	}

	return &api.AuthorizationRequest{}, fmt.Errorf("not found")
}

type redirectRequest struct {
	RedirectTo string `json:"redirectTo,omitempty"`
}

func getAuthorizationEndpointFromURL(u *url.URL) *api.AuthorizationRequest {
	endpoint := &api.AuthorizationRequest{
		URL: u,
	}

	for k, v := range u.Query() {
		switch k {
		case "client_id":
			if len(v) > 0 {
				endpoint.ClientID = v[0]
			}
		case "audience":
			if len(v) > 0 {
				endpoint.Audience = v[0]
			}
		case "scope":
			endpoint.Scopes = v
		}

	}

	return endpoint
}

// HasExternalAuthProvider checks if there is an external OAUTH2 provider is configured in the tenant
// Note: This does not require the client to be authenticated
func (c *Client) HasExternalAuthProvider(ctx context.Context) (loginOption *jsonmodels.LoginOption, found bool, err error) {
	options := c.LoginOptions.List(ctx, loginoptions.ListOptions{})
	if options.Err != nil {
		return nil, found, options.Err
	}

	for option := range op.Iter(options) {
		if strings.EqualFold(option.Type(), authentication.LoginTypeOAuth2) {
			loginOption = &option
			found = true
			break
		}
	}
	return
}

// AuthorizeWithDeviceFlow authorize the client using the OAuth2 Device Authorization Flow (the Auth provider must support it)
func (c *Client) AuthorizeWithDeviceFlow(ctx context.Context, initRequest string, auth_endpoints oauth2_api.AuthEndpoints, displayFunc device.DeviceCodeFunc) (*api.AccessToken, error) {
	// Create a new client which uses the given certificate
	// Use similar setting as the main client for consistency
	// tr := c.Client.Transport()
	// skipVerify := false
	// if httpTr, ok := tr.(*http.Transport); ok && httpTr.TLSClientConfig != nil {
	// 	skipVerify = httpTr.TLSClientConfig.InsecureSkipVerify
	// }

	httpClient := c.Client.Clone(context.Background()).Client()

	// httpClient := NewHTTPClient(
	// 	WithInsecureSkipVerify(skipVerify),
	// )
	endpoint, err := getAuthorizationRequest(ctx, httpClient, initRequest)
	if err != nil {
		return nil, err
	}

	scopes := make([]string, 0, len(auth_endpoints.Scopes))
	scopes = append(scopes, auth_endpoints.Scopes...)
	if len(scopes) == 0 {
		scopes = append(scopes, endpoint.Scopes...)
	}

	if auth_endpoints.TokenURL == "" || auth_endpoints.DeviceAuthorizationURL == "" {
		// Try detecting the endpoints via the open-id configuration endpoint
		openIDConfig := &api.OpenIDConfiguration{}

		if auth_endpoints.OpenIDConfigurationURL == "" {
			auth_endpoints.OpenIDConfigurationURL = api.GetOpenIDConnectConfigurationURL(endpoint.URL)
		}

		if err := api.GetOpenIDConfiguration(ctx, httpClient, endpoint.URL, auth_endpoints.OpenIDConfigurationURL, openIDConfig); err != nil {
			return nil, fmt.Errorf("%w. %w", ErrSSOInvalidConfiguration, err)
		} else {
			slog.Info("Found OpenID Connect configuration", "url", auth_endpoints.OpenIDConfigurationURL, "config", openIDConfig)
			if auth_endpoints.TokenURL == "" {
				auth_endpoints.TokenURL = openIDConfig.TokenEndpoint
			}
			if auth_endpoints.DeviceAuthorizationURL == "" {
				auth_endpoints.DeviceAuthorizationURL = openIDConfig.DeviceAuthorizationEndpoint
			}
		}

		// Add default scope if none are defined, as microsoft generally requires at least one scope
		if len(scopes) == 0 && len(openIDConfig.ScopesSupported) > 0 {
			slog.Info("Adding default scope", "value", openIDConfig.ScopesSupported[0])
			scopes = append(scopes, openIDConfig.ScopesSupported[0])
		}
	}

	deviceCodeURL := api.GetEndpointUrl(endpoint.URL, auth_endpoints.DeviceAuthorizationURL)
	requestCodeOptions := append([]api.AuthRequestEditorFn{}, auth_endpoints.AuthRequestOptions...)
	requestCodeOptions = append(requestCodeOptions, api.WithAudience(endpoint.Audience))
	slog.Info("Requesting device code", "url", deviceCodeURL, "client_id", endpoint.ClientID, "scopes", scopes)
	code, err := device.RequestCode(httpClient, deviceCodeURL, endpoint.ClientID, scopes, requestCodeOptions...)
	if err != nil {
		return nil, err
	}

	if displayFunc == nil {
		displayFunc = device.DeviceCodeOnConsole(os.Stderr)
	}

	if displayErr := displayFunc(code); displayErr != nil {
		return nil, displayErr
	}

	accessToken, err := device.Wait(context.TODO(), httpClient, api.GetEndpointUrl(endpoint.URL, auth_endpoints.TokenURL), device.WaitOptions{
		ClientID:   endpoint.ClientID,
		DeviceCode: code,
	})
	if err != nil {
		return nil, err
	}

	// Update client auth
	slog.Info("Using token from device flow")
	c.SetAuth(authentication.AuthOptions{
		Token: accessToken.Token,
	})

	return accessToken, nil
}
