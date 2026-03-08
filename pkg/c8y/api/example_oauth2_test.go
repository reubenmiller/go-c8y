package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	api "github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/devices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/oauth/clientcredentials"
)

// clientCredentialsSource fetches tokens from a standard OAuth2 token endpoint
// using the client_credentials grant. It does not depend on golang.org/x/oauth2.
//
// For production use you can swap this implementation for an
// oauth2.TokenSource adapter (see ExampleNewClient_withOAuth2Adapter).
type clientCredentialsSource struct {
	tokenURL     string
	clientID     string
	clientSecret string
	scopes       []string
	httpClient   *http.Client
}

func (s *clientCredentialsSource) Token() (*authentication.Token, error) {
	body := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {s.clientID},
		"client_secret": {s.clientSecret},
	}
	if len(s.scopes) > 0 {
		body.Set("scope", strings.Join(s.scopes, " "))
	}

	client := s.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.PostForm(s.tokenURL, body)
	if err != nil {
		return nil, fmt.Errorf("client credentials: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("client credentials: unexpected status %d: %s", resp.StatusCode, raw)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in,omitempty"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("client credentials: %w", err)
	}

	expiry := time.Time{}
	if payload.ExpiresIn > 0 {
		expiry = time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	}
	return &authentication.Token{
		AccessToken: payload.AccessToken,
		Expiry:      expiry,
	}, nil
}

// ExampleNewClient_withTokenSource shows how to connect to Cumulocity using a
// token obtained from an external OAuth2 provider via the client_credentials
// grant. Token renewal is automatic: the CachedTokenSource re-fetches a new
// token whenever the current one is (nearly) expired, and the retry middleware
// handles unexpected revocations (401 responses) by calling Invalidate and
// retrying once.
func ExampleNewClient_withTokenSource() {
	// Build the custom token source. Any implementation of
	// authentication.TokenSource works here — client credentials, device flow,
	// authorization code with PKCE, etc.
	rawSource := &clientCredentialsSource{
		tokenURL:     "https://idp.example.com/oauth2/token",
		clientID:     "my-client-id",
		clientSecret: "my-client-secret",
		scopes:       []string{"cumulocity"},
	}

	// Wrap with CachedTokenSource so every HTTP request does NOT hit the IdP.
	// The underlying source is called only when the cached token has expired.
	ts := authentication.NewCachedTokenSource(rawSource)

	client := api.NewClient(api.ClientOptions{
		BaseURL: "https://tenant.cumulocity.com",
		Auth: authentication.AuthOptions{
			TokenSource: ts,
		},
	})

	alarmCollection := client.Alarms.List(context.Background(), alarms.ListOptions{})
	if alarmCollection.Err != nil {
		panic(alarmCollection.Err)
	}

	for alarm := range jsondoc.DecodeIter[map[string]any](alarmCollection.Data.Iter()) {
		slog.Info("alarm", "alarm", alarm)
	}
}

// ExampleNewClient_withTokenSourceFunc shows the minimal path: supply a plain
// function via TokenSourceFunc when you already have a token-fetch helper and
// do not need explicit cache control.
func ExampleNewClient_withTokenSourceFunc() {
	getToken := func() (string, time.Time, error) {
		// Replace with your real token-fetch logic.
		return "my-token", time.Now().Add(time.Hour), nil
	}

	client := api.NewClient(api.ClientOptions{
		BaseURL: "https://tenant.cumulocity.com",
		Auth: authentication.AuthOptions{
			TokenSource: authentication.NewCachedTokenSource(
				authentication.TokenSourceFunc(func() (*authentication.Token, error) {
					tok, expiry, err := getToken()
					if err != nil {
						return nil, err
					}
					return &authentication.Token{AccessToken: tok, Expiry: expiry}, nil
				}),
			),
		},
	})

	_ = client
}

// ExampleNewClient_withOAuth2Adapter demonstrates how to integrate the
// standard golang.org/x/oauth2 library as a TokenSource. The adapter is a
// one-method wrapper that translates between the two Token types.
//
// To use this pattern add the dependency:
//
//	go get golang.org/x/oauth2
//
// then write:
//
//	import (
//	    "golang.org/x/oauth2"
//	    "golang.org/x/oauth2/clientcredentials"
//	)
//
//	type oauth2Adapter struct{ src oauth2.TokenSource }
//
//	func (a *oauth2Adapter) Token() (*authentication.Token, error) {
//	    t, err := a.src.Token() // handles caching & refresh internally
//	    if err != nil {
//	        return nil, err
//	    }
//	    return &authentication.Token{
//	        AccessToken: t.AccessToken,
//	        Expiry:      t.Expiry,
//	    }, nil
//	}
//
//	cfg := &clientcredentials.Config{
//	    ClientID:     "my-client-id",
//	    ClientSecret: "my-client-secret",
//	    TokenURL:     "https://idp.example.com/oauth2/token",
//	    Scopes:       []string{"cumulocity"},
//	}
//
//	client := api.NewClient(api.ClientOptions{
//	    BaseURL: "https://tenant.cumulocity.com",
//	    Auth: authentication.AuthOptions{
//	        // x/oauth2 already caches and refreshes internally, so no need to
//	        // wrap with CachedTokenSource.
//	        TokenSource: &oauth2Adapter{src: cfg.TokenSource(ctx)},
//	    },
//	})
func ExampleNewClient_withOAuth2Adapter() {
	// Stub so the example appears in the docs. The real code is shown above
	// in the function comment.
	_ = api.NewClient(api.ClientOptions{BaseURL: "https://tenant.cumulocity.com"})
}

// ExampleNewClient_withClientCredentials shows the minimal setup for the
// OAuth 2.0 client_credentials grant using the dedicated
// [clientcredentials.Config]. This is a simpler alternative to writing a
// custom [authentication.TokenSource] struct.
func ExampleNewClient_withClientCredentials() {
	cfg := &clientcredentials.Config{
		TokenURL:     "https://example.auth0.com/oauth/token",
		ClientID:     os.Getenv("SSO_CLIENT_ID"),
		ClientSecret: os.Getenv("SSO_CLIENT_SECRET"),

		// Scopes: []string{"cumulocity"},
		ExtraParams: url.Values{
			"audience": {"cumulocity"},
		},
	}

	client := api.NewClient(api.ClientOptions{
		BaseURL: "https://example.cumulocity.com",
		Auth: authentication.AuthOptions{
			// CachedTokenSource only calls cfg.Token() when the cached
			// token is expired, avoiding a round-trip on every request.
			TokenSource: authentication.NewCachedTokenSource(cfg),
		},
	})
	result := client.Devices.List(context.Background(), devices.ListOptions{
		PaginationOptions: pagination.PaginationOptions{
			WithTotalElements: true,
		},
	})
	fmt.Printf("Token: %s\n", client.Auth.Token)

	if result.Err != nil {
		panic(result.Err)
	}
	fmt.Printf("Total: %d\n", result.TotalElements())
}
