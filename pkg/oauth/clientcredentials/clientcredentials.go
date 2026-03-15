// Package clientcredentials implements an OAuth 2.0 client_credentials grant
// token source compatible with [authentication.TokenSource].
//
// Each call to [Config.Token] issues a fresh token request. Wrap with
// [authentication.NewCachedTokenSource] to avoid a round-trip on every request.
package clientcredentials

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
)

// Config holds the parameters for an OAuth 2.0 client_credentials grant.
// Config itself implements [authentication.TokenSource]; each Token() call
// makes a fresh network request. Wrap with [authentication.NewCachedTokenSource]
// to avoid unnecessary round-trips.
type Config struct {
	// TokenURL is the OAuth 2.0 token endpoint (e.g. returned by OIDC discovery
	// or extracted from the Cumulocity SSO login option's tokenRequest.url).
	TokenURL string

	// ClientID is the OAuth 2.0 client identifier.
	ClientID string

	// ClientSecret is the OAuth 2.0 client secret.
	ClientSecret string

	// Scopes is an optional list of OAuth 2.0 scopes to request.
	Scopes []string

	// HTTPClient overrides the HTTP client used for token requests.
	// Defaults to http.DefaultClient when nil.
	HTTPClient *http.Client

	// ExtraParams holds additional form parameters to include in the token
	// request (e.g. "audience", "resource"). These are merged with the
	// standard grant_type/client_id/client_secret/scope parameters.
	ExtraParams url.Values
}

// Token fetches a fresh bearer token from the configured token endpoint using
// the client_credentials grant. It satisfies [authentication.TokenSource].
//
// The caller is responsible for caching; this method always makes a network
// request. Use [authentication.NewCachedTokenSource] to add caching.
func (c *Config) Token() (*authentication.Token, error) {
	body := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
	}
	if len(c.Scopes) > 0 {
		body.Set("scope", strings.Join(c.Scopes, " "))
	}
	for k, vs := range c.ExtraParams {
		for _, v := range vs {
			body.Add(k, v)
		}
	}

	hc := c.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}

	resp, err := hc.PostForm(c.TokenURL, body)
	if err != nil {
		return nil, fmt.Errorf("client_credentials: token request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("client_credentials: unexpected status %d from %s: %s",
			resp.StatusCode, c.TokenURL, raw)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in,omitempty"`
		TokenType   string `json:"token_type,omitempty"`
		Error       string `json:"error,omitempty"`
		ErrorDesc   string `json:"error_description,omitempty"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("client_credentials: failed to parse token response: %w", err)
	}
	if payload.Error != "" {
		return nil, fmt.Errorf("client_credentials: %s: %s", payload.Error, payload.ErrorDesc)
	}
	if payload.AccessToken == "" {
		return nil, fmt.Errorf("client_credentials: no access_token in response from %s", c.TokenURL)
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
