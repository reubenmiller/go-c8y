package api

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/logintokens"
)

// contextTokenRetryCooldown is how long the client waits before re-attempting
// a token exchange for a tenant/user after a failed attempt. During the
// cooldown, requests for that tenant/user fall back to basic auth without
// paying a login round-trip.
const contextTokenRetryCooldown = 5 * time.Minute

// contextTokenEntry holds the cached token source for one tenant/user pair.
// The mutex also serialises concurrent first-time fetches for the same pair so
// parallel requests do not stampede the login endpoint.
type contextTokenEntry struct {
	mu               sync.Mutex
	password         string
	source           *authentication.CachedTokenSource
	unavailableUntil time.Time
}

// SetContextTokenExchange enables or disables exchanging per-request basic
// credentials (see WithAuth / WithServiceUser) for OAI-Secure bearer tokens.
//
// When enabled, the first request for a given tenant/user pays one login
// round-trip; the token is then cached and refreshed automatically on expiry
// or after a 401. When the exchange fails (e.g. the tenant does not support
// OAI-Secure), the request falls back to basic auth and the exchange is not
// re-attempted for that tenant/user for a cooldown period.
//
// This is useful in MULTI_TENANT microservices where sending basic service
// user credentials on every downstream request is undesirable.
func (c *Client) SetContextTokenExchange(enable bool) {
	c.contextTokenExchange.Store(enable)
	if !enable {
		c.contextTokens.Clear()
	}
}

// ContextTokenExchangeEnabled reports whether per-request credentials are
// exchanged for bearer tokens.
func (c *Client) ContextTokenExchangeEnabled() bool {
	return c.contextTokenExchange.Load()
}

func contextTokenKey(auth authentication.AuthOptions) string {
	return authentication.JoinTenantUser(auth.Tenant, auth.Username)
}

// contextTokenFor returns a bearer token for the given per-request credentials,
// using a cached per-tenant token source. It returns an empty string when the
// exchange is disabled, in cooldown after a failure, or fails — in which case
// the caller falls back to basic auth.
func (c *Client) contextTokenFor(auth authentication.AuthOptions) string {
	if c == nil || !c.contextTokenExchange.Load() {
		return ""
	}

	v, _ := c.contextTokens.LoadOrStore(contextTokenKey(auth), &contextTokenEntry{})
	entry := v.(*contextTokenEntry)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Detect credential rotation (e.g. a service user recreated after an
	// unsubscribe/subscribe cycle) and reset the cached state.
	if entry.password != auth.Password {
		entry.password = auth.Password
		entry.source = nil
		entry.unavailableUntil = time.Time{}
	}

	if !entry.unavailableUntil.IsZero() && time.Now().Before(entry.unavailableUntil) {
		return ""
	}

	if entry.source == nil {
		creds := auth
		entry.source = authentication.NewCachedTokenSource(authentication.TokenSourceFunc(func() (*authentication.Token, error) {
			return c.fetchTokenFor(creds)
		}))
	}

	tok, err := entry.source.Token()
	if err != nil || tok == nil || tok.AccessToken == "" {
		slog.Debug("Context token exchange failed, falling back to basic auth", "tenant", auth.Tenant, "err", err)
		entry.unavailableUntil = time.Now().Add(contextTokenRetryCooldown)
		return ""
	}
	return tok.AccessToken
}

// invalidateContextToken clears the cached token for the given credentials so
// the next request fetches a fresh one (e.g. after a 401).
func (c *Client) invalidateContextToken(auth authentication.AuthOptions) {
	if v, ok := c.contextTokens.Load(contextTokenKey(auth)); ok {
		entry := v.(*contextTokenEntry)
		entry.mu.Lock()
		if entry.source != nil {
			entry.source.Invalidate()
		}
		entry.mu.Unlock()
	}
}

// fetchTokenFor exchanges the given basic credentials for an OAI-Secure bearer
// token. Unlike fetchToken it does not use or modify the client's own auth
// state, so it is safe to call for any tenant/user.
func (c *Client) fetchTokenFor(auth authentication.AuthOptions) (*authentication.Token, error) {
	// A fresh context (no per-request credentials, token source skipped) so the
	// login call cannot recurse into the context-auth or token-source middleware.
	ctx := ctxhelpers.WithSkipTokenSource(context.Background())
	tok := c.LoginTokens.Create(ctx, logintokens.CreateTokenOptions{
		Tenant:    auth.Tenant,
		Username:  auth.Username,
		Password:  auth.Password,
		GrantType: logintokens.GrantTypePassword,
	})
	if tok.IsError() {
		return nil, tok.Err
	}
	raw := tok.Data.AccessToken()
	if raw == "" {
		return nil, errors.New("token exchange returned an empty access token")
	}
	// Default expiry — C8Y tokens are typically 1 hour; use 55 min as a buffer.
	expiry := time.Now().Add(55 * time.Minute)
	if claims, parseErr := authentication.ParseToken(raw); parseErr == nil && claims.ExpiresAt != nil {
		expiry = claims.ExpiresAt.Time
	}
	return &authentication.Token{AccessToken: raw, Expiry: expiry}, nil
}
