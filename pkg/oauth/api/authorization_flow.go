package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type AuthorizationCodeFunc func(string, AuthorizationCodeOptions) error

func AuthorizationFlowOnConsole(w io.Writer) AuthorizationCodeFunc {
	return func(u string, auth AuthorizationCodeOptions) error {
		var err error
		_, err = fmt.Fprintf(w, `
Please visit this URL in your browser:

  %s

After authorizing, you'll be redirected to:

  %s

`, u, auth.RedirectURI)
		return err
	}
}

// TokenResponse represents the OAuth2 token response from Cumulocity
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type AuthorizationCodeOptions struct {
	BaseURL     string
	Tenant      string
	ClientID    string
	Scopes      []string
	State       string
	RedirectURI string

	TokenURL string

	DisplayFunc AuthorizationCodeFunc
}

// GetAuthorizationURL generates the OAuth2 authorization URL for the authorization code flow
func GetAuthorizationURL(opts AuthorizationCodeOptions) (string, error) {
	u, err := url.Parse(opts.BaseURL)
	if err != nil {
		return "", err
	}

	params := u.Query()

	// clear existing
	u.RawQuery = ""

	// Build query parameters
	if opts.Tenant != "" {
		params.Set("tenant_id", opts.Tenant)
	}
	if opts.ClientID != "" {
		params.Set("client_id", opts.ClientID)
	}
	if opts.State != "" {
		params.Set("state", opts.State)
	}
	params.Set("response_type", "code")
	params.Set("redirect_uri", opts.RedirectURI)
	if len(opts.Scopes) > 0 {
		params.Set("scope", strings.Join(opts.Scopes, " "))
	}

	u.RawQuery = params.Encode()
	return u.String(), nil
}

// PerformAuthorizationCodeFlow performs the complete OAuth2 authorization code flow for CLI
func PerformAuthorizationCodeFlow(ctx context.Context, opts AuthorizationCodeOptions) (string, error) {
	// Generate authorization URL
	authURL, err := GetAuthorizationURL(opts)
	if err != nil {
		return "", err
	}

	if opts.DisplayFunc == nil {
		opts.DisplayFunc = AuthorizationFlowOnConsole(os.Stderr)
	}

	if displayErr := opts.DisplayFunc(authURL, opts); displayErr != nil {
		return "", displayErr
	}

	// Start local server to capture the code
	code, err := StartLocalServerForCode(ctx, opts.RedirectURI)
	if err != nil {
		return "", fmt.Errorf("failed to capture authorization code: %w", err)
	}

	return code, nil
}

// StartLocalServerForCode starts a local HTTP server to capture the authorization code
func StartLocalServerForCode(ctx context.Context, redirectURI string) (string, error) {
	// Parse the redirect URI to get port
	parsedURI, err := url.Parse(redirectURI)
	if err != nil {
		return "", fmt.Errorf("invalid redirect URI: %w", err)
	}

	port := parsedURI.Port()
	if port == "" {
		port = "80"
	}

	// Channel to receive the authorization code
	codeChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// Start local server

	mux := http.NewServeMux()
	server := &http.Server{Addr: ":" + port, Handler: mux}
	mux.HandleFunc(parsedURI.Path, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errorChan <- fmt.Errorf("no authorization code in request")
			return
		}

		// Return success page to user
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
	<title>🎉 Authorization Successful! 🎉</title>
	<style>
		body {
			font-family: 'Segoe UI', Arial, sans-serif;
			background: linear-gradient(135deg, #e0c3fc 0%, #8ec5fc 100%);
			color: #333;
			text-align: center;
			padding-top: 60px;
		}
		.emoji {
			font-size: 4rem;
			margin-bottom: 20px;
			animation: bounce 1s infinite alternate;
		}
		@keyframes bounce {
			to { transform: translateY(-20px); }
		}
		.card {
			background: #fff;
			display: inline-block;
			padding: 32px 48px;
			border-radius: 16px;
			box-shadow: 0 4px 24px rgba(0,0,0,0.08);
			margin-top: 20px;
		}
		.close {
			margin-top: 24px;
			font-size: 1.1rem;
			color: #555;
		}
	</style>
</head>
<body>
	<div class="card">
		<div class="emoji">go-c8y-cli 💻</div>
		<h1>OAuth2 Authorization!</h1>
		<p>Authorization was <b>successful</b>.<br>
		You can now return to your terminal to continue.</p>
		<div class="close">You may safely close this window. 👋</div>
	</div>
</body>
</html>
`)

		codeChan <- code
	})

	// Start server in background
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errorChan <- err
		}
	}()

	defer server.Shutdown(context.Background())

	// Wait for either code or error
	select {
	case code := <-codeChan:
		return code, nil
	case err := <-errorChan:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
