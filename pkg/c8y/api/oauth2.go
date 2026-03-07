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
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
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

					// TODO: allow users to pass this value
					// params.Set("originUri", redirectURL)

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
	if initRequest == "" {
		loginOption, found, err := c.HasExternalAuthProvider(context.Background())
		if err != nil {
			// error getting details
			return nil, err
		}
		if !found {
			// no external auth provider
			return nil, core.ErrNoAuth2Provider
		}
		initRequest = loginOption.InitRequest()
	}

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
			slog.Debug("Found OpenID Connect configuration", "url", auth_endpoints.OpenIDConfigurationURL, "config", openIDConfig)
			if auth_endpoints.TokenURL == "" {
				auth_endpoints.TokenURL = openIDConfig.TokenEndpoint
			}
			if auth_endpoints.DeviceAuthorizationURL == "" {
				auth_endpoints.DeviceAuthorizationURL = openIDConfig.DeviceAuthorizationEndpoint
			}
		}

		// Add default scope if none are defined, as microsoft generally requires at least one scope
		if len(scopes) == 0 && len(openIDConfig.ScopesSupported) > 0 {
			slog.Debug("Adding default scope", "value", openIDConfig.ScopesSupported[0])
			scopes = append(scopes, openIDConfig.ScopesSupported[0])
		}
	}

	deviceCodeURL := api.GetEndpointUrl(endpoint.URL, auth_endpoints.DeviceAuthorizationURL)
	requestCodeOptions := append([]api.AuthRequestEditorFn{}, auth_endpoints.AuthRequestOptions...)
	requestCodeOptions = append(requestCodeOptions, api.WithAudience(endpoint.Audience))
	slog.Debug("Requesting device code", "url", deviceCodeURL, "client_id", endpoint.ClientID, "scopes", scopes)
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
	slog.Debug("Using token from device flow")
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

	// CallbackURL is the full redirect URI that the SSO provider will redirect
	// the browser to after a successful authentication. It must exactly match
	// a URI pre-registered in your SSO provider configuration.
	//
	// Accepted forms (all equivalent in terms of where the local server listens
	// and which path it registers the handler on):
	//
	//   http://127.0.0.1:5001/callback  – explicit scheme, host, port, path
	//   127.0.0.1:5001/callback         – scheme inferred as http
	//   127.0.0.1:5001                  – path defaults to /callback
	//
	// Defaults to "http://127.0.0.1:5001/callback" when empty.
	CallbackURL string

	OriginURL string

	// SuccessPage is the full HTML body returned to the browser after a
	// successful authentication.  When empty the built-in Cumulocity-branded
	// page (browser_success.html) is used.
	SuccessPage string
}

// parseBrowserCallbackURL normalises the CallbackURL field into the three
// pieces needed by AuthorizeWithBrowserFlow:
//
//   - listenAddr  – "host:port" string passed to net.Listen
//   - path        – URL path to register the HTTP handler on (e.g. "/callback")
//   - callbackURL – the canonical, fully-qualified redirect URI sent to
//     the SSO provider (always http://host:port/path)
//
// Input forms handled:
//
//	http://127.0.0.1:5001/callback  -> ("127.0.0.1:5001", "/callback", "http://127.0.0.1:5001/callback")
//	127.0.0.1:5001/mypath           -> ("127.0.0.1:5001", "/mypath",   "http://127.0.0.1:5001/mypath")
//	127.0.0.1:5001                  -> ("127.0.0.1:5001", "/callback", "http://127.0.0.1:5001/callback")
//	"" (empty)                      -> ("127.0.0.1:5001", "/callback", "http://127.0.0.1:5001/callback")
func parseBrowserCallbackURL(raw string) (listenAddr, path, callbackURL string, err error) {
	const defaultAddr = "127.0.0.1:5001"
	const defaultPath = "/callback"

	if raw == "" {
		return defaultAddr, defaultPath,
			"http://" + defaultAddr + defaultPath, nil
	}

	// Ensure there is a scheme so url.Parse interprets host and path correctly.
	// Without a scheme, url.Parse treats "host:port/path" as scheme=host,
	// opaque="port/path" which is not what we want.
	toParse := raw
	if !strings.Contains(raw, "://") {
		toParse = "http://" + raw
	}

	u, parseErr := url.Parse(toParse)
	if parseErr != nil {
		err = fmt.Errorf("browser flow: invalid CallbackURL %q: %w", raw, parseErr)
		return
	}
	if u.Host == "" {
		err = fmt.Errorf("browser flow: CallbackURL %q has no host:port", raw)
		return
	}

	listenAddr = u.Host
	path = u.Path
	if path == "" {
		path = defaultPath
	}
	callbackURL = "http://" + listenAddr + path
	return
}

// DeviceFlowOptions configures the OAuth2 Device Authorization flow used by
// LoginWithOptions when LoginMethodOAuth2DeviceFlow is selected.
type DeviceFlowOptions struct {
	// AuthEndpoints overrides the auto-discovered OAuth2 endpoint URLs (token
	// URL, device-authorization URL, OpenID configuration URL, etc.).
	// Leave zero-valued to have the client discover them automatically from the
	// tenant's OpenID Connect configuration endpoint.
	AuthEndpoints oauth2_api.AuthEndpoints

	// DisplayFunc is called with the device authorization code so the user can
	// visit the verification URL and enter the code.
	// Defaults to device.DeviceCodeOnConsole(os.Stderr) when nil.
	DisplayFunc device.DeviceCodeFunc
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
	if initRequest == "" {
		loginOption, found, err := c.HasExternalAuthProvider(context.Background())
		if err != nil {
			// error getting details
			return nil, err
		}
		if !found {
			// no external auth provider
			return nil, core.ErrNoAuth2Provider
		}
		initRequest = loginOption.InitRequest()
	}

	if opts.OpenBrowser == nil {
		opts.OpenBrowser = DefaultBrowserOpen
	}

	listenAddr, callbackPath, callbackURL, err := parseBrowserCallbackURL(opts.CallbackURL)
	if err != nil {
		return nil, err
	}

	// Start local callback server.
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("browser flow: start callback listener: %w", err)
	}

	codeCh := make(chan string, 1)
	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
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
		successHTML := opts.SuccessPage
		if successHTML == "" {
			successHTML = browserSuccessPage
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, successHTML)
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
	slog.Debug("Browser flow: opening IdP authorization URL", "url", idpAuthURL)

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
		tokenClient.SetDebug(c.debugEnabled)
		tok := tokenClient.LoginTokens.Create(ctx, logintokens.CreateTokenOptions{
			GrantType:     logintokens.GrantTypeAuthorizationCode,
			Code:          code,
			RequestOrigin: callbackURL,
		})

		if tok.Err != nil {
			return nil, fmt.Errorf("browser flow: token exchange: %w", tok.Err)
		}
		accessToken := tok.Data.AccessToken()
		if accessToken == "" {
			return nil, fmt.Errorf("browser flow: token exchange returned empty access token")
		}
		c.SetAuth(authentication.AuthOptions{Token: accessToken})
		return &api.AccessToken{Token: accessToken}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
