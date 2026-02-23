package api

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/loginoptions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/logintokens"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/pkg/oauth/api"
	oauth2_api "github.com/reubenmiller/go-c8y/pkg/oauth/api"
	"github.com/reubenmiller/go-c8y/pkg/oauth/device"
	"github.com/tidwall/gjson"
)

var ErrSSOInvalidConfiguration = errors.New("invalid sso configuration")

// GetLoginOptions returns the login options available for the tenant
func getAuthorizationRequest(ctx context.Context, client *http.Client, oauthUrl string, redirectURL string) (*api.AuthorizationRequest, error) {
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
					if redirectURL != "" {
						params.Set("redirect_uri", redirectURL)
					}
					u.RawQuery = params.Encode()

					return getAuthorizationEndpointFromURL(u), nil
				}
			}
		}
	}

	return &api.AuthorizationRequest{}, fmt.Errorf("not found")
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

	httpClient := c.Client.Clone(context.Background()).Client()
	endpoint, err := getAuthorizationRequest(ctx, httpClient, initRequest, "")
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

// BrowserOpenFunc is called with the IdP authorization URL so the caller can
// open it in a browser (or print it for the user to open manually).
type BrowserOpenFunc func(url string) error

// DefaultBrowserOpen opens url in the system default browser.
func DefaultBrowserOpen(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "cmd", []string{"/c", "start", url}
	default: // linux, bsd, …
		cmd, args = "xdg-open", []string{url}
	}
	return exec.Command(cmd, args...).Start()
}

//go:embed browser_success.html
var browserSuccessPage string

// BrowserFlowOptions controls the behaviour of AuthorizeWithBrowserFlow.
type BrowserFlowOptions struct {
	// OpenBrowser is called with the IdP authorization URL.  Defaults to
	// DefaultBrowserOpen when nil.
	OpenBrowser BrowserOpenFunc

	// ListenAddr sets the local TCP listen address for the callback server,
	// e.g. "127.0.0.1:5001".  Defaults to "127.0.0.1:5001".
	//
	// NOTE: The full callback URL (http://localhost:<port>/callback) must be
	// pre-registered as an allowed redirect URI in your SSO provider.  Use a
	// fixed port that matches your provider's configuration.
	ListenAddr string
}

// AuthorizeWithBrowserFlow performs the OAuth2 Authorization Code flow
// interactively from a CLI:
//
//  1. A local HTTP callback server is started on a random port.
//  2. The Cumulocity initRequest URL is called (with the callback URL embedded)
//     to obtain the external IdP authorization URL.
//  3. The browser is opened to the IdP login page via opts.OpenBrowser.
//  4. The user authenticates; Cumulocity redirects the browser to the local
//     callback server with an authorization code.
//  5. The code is exchanged for a Cumulocity access token via
//     POST /tenant/oauth/token (grant_type=AUTHORIZATION_CODE).
//  6. The client's auth is updated with the new token.
//
// The method blocks until the code is received or ctx is cancelled.
func (c *Client) AuthorizeWithBrowserFlow(ctx context.Context, initRequest string, opts BrowserFlowOptions) (*api.AccessToken, error) {
	if opts.OpenBrowser == nil {
		opts.OpenBrowser = DefaultBrowserOpen
	}
	listenAddr := opts.ListenAddr
	if listenAddr == "" {
		listenAddr = "127.0.0.1:5001"
	}

	// Start local callback server.
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("browser flow: start callback listener: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	codeCh := make(chan string, 1)
	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		errParam := r.URL.Query().Get("error")
		if errParam != "" {
			msg := r.URL.Query().Get("error_description")
			if msg == "" {
				msg = errParam
			}
			http.Error(w, "SSO error: "+msg, http.StatusBadRequest)
			codeCh <- ""
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code parameter", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, browserSuccessPage)
		codeCh <- code
		go func() { _ = srv.Shutdown(context.Background()) }()
	})

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Browser flow: callback server error", "err", err)
		}
	}()
	defer srv.Shutdown(ctx) //nolint:errcheck

	// Fetch IdP authorization URL; callbackURL is embedded in the request so
	// Cumulocity knows where to redirect the browser after the SSO exchange.
	httpClient := c.Client.Clone(context.Background()).Client()
	authReq, err := getAuthorizationRequest(ctx, httpClient, initRequest, callbackURL)
	if err != nil {
		return nil, fmt.Errorf("browser flow: get authorization URL: %w", err)
	}
	idpAuthURL := authReq.URL.String()
	slog.Info("Browser flow: opening IdP authorization URL", "url", idpAuthURL)

	if err := opts.OpenBrowser(idpAuthURL); err != nil {
		slog.Warn("Browser flow: could not open browser", "err", err)
		fmt.Fprintf(os.Stderr, "Open this URL in your browser:\n  %s\n", idpAuthURL)
	}

	// Wait for the authorization code.
	select {
	case code := <-codeCh:
		if code == "" {
			return nil, fmt.Errorf("browser flow: SSO callback returned an error")
		}
		tokenClient := NewClient(ClientOptions{BaseURL: c.Client.BaseURL()})
		tokenClient.Client.SetCookie(&http.Cookie{
			Name:  "REQUEST_ORIGIN",
			Value: callbackURL,
		})

		tok := tokenClient.LoginTokens.Create(ctx, logintokens.CreateTokenOptions{
			GrantType: logintokens.GrantTypeAuthorizationCode,
			Code:      code,
		})
		if tok.Err != nil {
			return nil, fmt.Errorf("browser flow: token exchange: %w", tok.Err)
		}
		accessToken := tok.Data.AccessToken()
		if accessToken == "" {
			return nil, fmt.Errorf("browser flow: token exchange returned empty access token")
		}
		c.SetAuth(authentication.AuthOptions{Token: accessToken})
		slog.Info("Browser flow: token obtained")
		return &api.AccessToken{Token: accessToken}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
