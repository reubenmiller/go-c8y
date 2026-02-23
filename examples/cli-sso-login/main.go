// Package main demonstrates the Cumulocity SSO Authorization Code flow from a
// CLI using a local HTTP callback server.
//
// # How it works
//
//  1. A local HTTP server is started on a random port.
//  2. The Cumulocity login options are fetched (unauthenticated) to find the
//     SSO provider and its "initRequest" URL.
//  3. The initRequest URL is called by the CLI's own HTTP client, NOT the
//     browser.  Two cookies are attached to that request:
//     - REQUEST_ORIGIN  – Cumulocity base URL (required for CORS validation)
//     - REDIRECT_URI    – our local callback URL (http://localhost:PORT/callback)
//     Cumulocity uses the REDIRECT_URI to know where to send the browser after
//     the SSO exchange has completed.
//  4. The 302 Location from that request is the IdP's authorization URL,
//     which already embeds the correct redirect_uri pointing back to Cumulocity
//     and a state that encodes the REDIRECT_URI for later use.
//  5. The browser is opened to the IdP authorization URL.
//  6. The user authenticates; the IdP redirects the browser to
//     <C8Y_HOST>/tenant/oauth?code=<CODE>.  Cumulocity exchanges the code with
//     the IdP, sets auth cookies in the browser, then (because of the
//     REDIRECT_URI stored in the state) redirects the browser to
//     http://localhost:PORT/callback?code=<C8Y_CODE>.
//  7. The local server captures the code from the query string.
//  8. The code is exchanged via POST /tenant/oauth/token
//     (grant_type=AUTHORIZATION_CODE) to obtain a Cumulocity access token.
//  9. The access token is used to create an authenticated API client and a
//     test call is made.
//
// Required environment variable:
//
//	C8Y_BASEURL – e.g. https://mytenant.cumulocity.com
//
// References:
//   - https://cumulocity.com/api/core/#section/Authentication/SSO
//   - https://www.rfc-editor.org/rfc/rfc6749#section-4.1
package main

import (
	"context"
	"encoding/json"
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
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/currenttenant"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/logintokens"
)

const callbackPath = "/callback"

// startCallbackServer starts a local HTTP server on a random available port.
// It returns the server, the full callback URL (http://localhost:PORT/callback),
// and a channel that will receive the authorization code exactly once.
// The server shuts itself down after the first successful callback.
func startCallbackServer() (*http.Server, string, <-chan string, error) {
	// Bind to an OS-assigned port on loopback.
	ln, err := net.Listen("tcp", "127.0.0.1:5001")
	if err != nil {
		return nil, "", nil, fmt.Errorf("start callback listener: %w", err)
	}

	port := ln.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d%s", port, callbackPath)

	codeCh := make(chan string, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		errParam := r.URL.Query().Get("error")

		if errParam != "" {
			msg := r.URL.Query().Get("error_description")
			if msg == "" {
				msg = errParam
			}
			http.Error(w, "SSO error: "+msg, http.StatusBadRequest)
			codeCh <- "" // unblock the waiter
			return
		}

		if code == "" {
			http.Error(w, "missing code parameter", http.StatusBadRequest)
			return
		}

		fmt.Fprintln(w, "<html><body><h2>Authentication successful - you may close this tab.</h2></body></html>")
		codeCh <- code

		// Shut the server down asynchronously so the response is flushed first.
		go func() {
			_ = srv.Shutdown(context.Background())
		}()
	})

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Callback server error", "err", err)
		}
	}()

	return srv, callbackURL, codeCh, nil
}

// fetchIdPAuthURL calls the Cumulocity initRequest URL (from the CLI's own
// HTTP client, not the browser) with the REDIRECT_URI and REQUEST_ORIGIN
// cookies attached.  Cumulocity stores the REDIRECT_URI in the OAuth2 state
// parameter so that, after the user authenticates with the IdP, the browser
// is eventually redirected back to our local callback server.
//
// The function returns the 302 Location, which is the IdP's authorization URL
// – this is the URL the user must open in their browser.
func fetchIdPAuthURL(ctx context.Context, initRequestURL, redirectURI, requestOrigin string) (string, error) {
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // capture the first redirect only
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, initRequestURL, nil)
	if err != nil {
		return "", fmt.Errorf("build initRequest: %w", err)
	}

	slog.Info("Auth settings.", "redirectURI", redirectURI)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("initRequest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		// Status code 200 is returned when the "Redirect to the user interface application" option is enabled
		// Use when the "redirect to user application" is enabled
		if b, err := io.ReadAll(resp.Body); err == nil {
			data := make(map[string]string)
			if err := json.Unmarshal(b, &data); err == nil {
				slog.Info("Auth URL.", "redirectTo", data["redirectTo"])

				// TODO: replace the redirect_uri query parameter with the redirectUri value
				u, err := url.Parse(data["redirectTo"])
				if err == nil {
					params := u.Query()
					params.Set("redirect_uri", redirectURI)
					u.RawQuery = params.Encode()
				}

				slog.Info("Modified url.", "url", u.String())
				return u.String(), nil
			}
		}
	} else if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		return "", fmt.Errorf("initRequest: unexpected status %d (want 3xx)", resp.StatusCode)
	}

	loc, err := resp.Location()
	if err != nil {
		return "", fmt.Errorf("initRequest: read Location: %w", err)
	}
	return loc.String(), nil
}

// openBrowser attempts to open url in the default system browser.
func openBrowser(url string) error {
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

func main() {
	ctx := context.Background()

	// -----------------------------------------------------------------------
	// 1. Read the base URL from the environment.
	// -----------------------------------------------------------------------
	baseURL := authentication.HostFromEnvironment()
	if baseURL == "" {
		fmt.Fprintln(os.Stderr, "error: C8Y_BASEURL is not set")
		os.Exit(1)
	}

	// -----------------------------------------------------------------------
	// 2. Create an unauthenticated client to discover the SSO login option.
	// -----------------------------------------------------------------------
	discoveryClient := api.NewClient(api.ClientOptions{BaseURL: baseURL})

	loginOption, found, err := discoveryClient.HasExternalAuthProvider(ctx)
	if err != nil {
		slog.Error("Failed to retrieve Cumulocity login options", "err", err)
		os.Exit(1)
	}
	if !found {
		slog.Error("No external OAuth2 SSO provider is configured on this Cumulocity tenant")
		os.Exit(1)
	}

	initRequest := loginOption.InitRequest()
	slog.Info("Found SSO login option", "initRequest", initRequest)

	// -----------------------------------------------------------------------
	// 3. Start the local callback server.
	// -----------------------------------------------------------------------
	srv, callbackURL, codeCh, err := startCallbackServer()
	if err != nil {
		slog.Error("Failed to start callback server", "err", err)
		os.Exit(1)
	}
	defer srv.Shutdown(ctx) //nolint:errcheck
	slog.Info("Callback server started", "url", callbackURL)

	// -----------------------------------------------------------------------
	// 4. Call initRequest (from CLI) with REDIRECT_URI + REQUEST_ORIGIN cookies
	//    to obtain the IdP authorization URL.
	// -----------------------------------------------------------------------
	idpAuthURL, err := fetchIdPAuthURL(ctx, initRequest, callbackURL, baseURL)
	if err != nil {
		slog.Error("Failed to get IdP authorization URL", "err", err)
		os.Exit(1)
	}

	slog.Info("IdP authorization URL obtained", "url", idpAuthURL)

	// -----------------------------------------------------------------------
	// 5. Open the browser so the user can authenticate.
	// -----------------------------------------------------------------------
	fmt.Fprintf(os.Stderr, "\n🌐 Opening browser for SSO login …\n")
	fmt.Fprintf(os.Stderr, "   If the browser does not open automatically, visit:\n   %s\n\n", idpAuthURL)
	if err := openBrowser(idpAuthURL); err != nil {
		slog.Warn("Could not open browser automatically", "err", err)
	}

	// -----------------------------------------------------------------------
	// 6. Wait for the authorization code to arrive at the callback server.
	// -----------------------------------------------------------------------
	fmt.Fprintln(os.Stderr, "⏳ Waiting for SSO callback …")
	select {
	case code := <-codeCh:
		if code == "" {
			slog.Error("SSO callback returned an error (see browser for details)")
			os.Exit(1)
		}
		slog.Info("Authorization code received")

		// -------------------------------------------------------------------
		// 7. Exchange the code for a Cumulocity access token.
		// -------------------------------------------------------------------
		fmt.Fprintln(os.Stderr, "🔑 Exchanging authorization code for access token …")
		tokenClient := api.NewClient(api.ClientOptions{BaseURL: baseURL})
		tokenClient.Client.SetDebug(true)
		tokenClient.Client.SetCookie(&http.Cookie{
			Name:  "REQUEST_ORIGIN",
			Value: callbackURL,
		})
		tok := tokenClient.LoginTokens.Create(ctx, logintokens.CreateTokenOptions{
			GrantType: logintokens.GrantTypeAuthorizationCode,
			Code:      code,
		})
		if tok.Err != nil {
			slog.Error("Token exchange failed", "err", tok.Err)
			os.Exit(1)
		}

		accessToken := tok.Data.AccessToken()
		if accessToken == "" {
			slog.Error("Token exchange returned an empty access token")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "✅ Access token obtained")

		// -------------------------------------------------------------------
		// 8. Create the authenticated client and make a verification call.
		// -------------------------------------------------------------------
		client := api.NewClient(api.ClientOptions{
			BaseURL: baseURL,
			Auth: authentication.AuthOptions{
				Token: accessToken,
			},
		})

		fmt.Fprintln(os.Stderr, "🔍 Verifying token against Cumulocity API …")
		result := client.Tenants.Current.Get(ctx, currenttenant.GetOptions{})
		if result.Err != nil {
			slog.Error("API call failed – the token may not be accepted by Cumulocity",
				"err", result.Err,
			)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "✅ Successfully authenticated via SSO Authorization Code flow\n")
		fmt.Fprintf(os.Stderr, "   Tenant: %s\n", result.Data.Name())
		fmt.Fprintf(os.Stderr, "   Domain: %s\n", result.Data.Domain())

	case <-time.After(5 * time.Minute):
		slog.Error("Timed out waiting for SSO callback")
		os.Exit(1)
	}
}
