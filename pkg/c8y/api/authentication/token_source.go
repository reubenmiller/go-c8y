package authentication

import (
	"sync"
	"time"
)

// Token is a bearer token with an optional expiry.
type Token struct {
	AccessToken string
	// Expiry is the time at which the token expires. A zero value means no
	// expiry information is available; the token is assumed to be long-lived.
	Expiry time.Time
}

// Valid reports whether t is non-empty and not within 10 seconds of expiry.
func (t *Token) Valid() bool {
	return t != nil && t.AccessToken != "" &&
		(t.Expiry.IsZero() || t.Expiry.After(time.Now().Add(10*time.Second)))
}

// TokenSource yields bearer tokens for use in HTTP requests.
// Implementations must be safe for concurrent use.
//
// The interface is intentionally compatible with golang.org/x/oauth2.TokenSource
// so that external OAuth2 token sources can be adapted with a thin wrapper.
type TokenSource interface {
	Token() (*Token, error)
}

// TokenSourceFunc adapts a plain function to TokenSource.
type TokenSourceFunc func() (*Token, error)

func (f TokenSourceFunc) Token() (*Token, error) { return f() }

// CachedTokenSource wraps a TokenSource and caches the token until it is (nearly)
// expired. It is safe for concurrent use.
//
// Use NewCachedTokenSource to construct one. Call Invalidate to force the next
// Token() call to fetch a fresh token regardless of expiry (e.g. after a 401).
type CachedTokenSource struct {
	mu     sync.Mutex
	base   TokenSource
	cached *Token
}

// NewCachedTokenSource returns a CachedTokenSource wrapping base.
func NewCachedTokenSource(base TokenSource) *CachedTokenSource {
	return &CachedTokenSource{base: base}
}

// Token returns the cached token if still valid, otherwise fetches a new one from base.
func (c *CachedTokenSource) Token() (*Token, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cached.Valid() {
		return c.cached, nil
	}
	tok, err := c.base.Token()
	if err != nil {
		return nil, err
	}
	c.cached = tok
	return tok, nil
}

// Invalidate clears the cached token, forcing the next Token() call to fetch a fresh one.
// Call this after receiving a 401 to ensure stale tokens are not reused.
func (c *CachedTokenSource) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cached = nil
}

// Seed pre-populates the cache with tok, avoiding an extra network round-trip on
// first use when the caller already holds a valid token (e.g. just after login).
func (c *CachedTokenSource) Seed(tok *Token) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cached = tok
}

// InvalidateToken clears the cache only when it still holds rejected, the token
// a server refused. When another caller has already replaced it with a new token,
// that token is kept. Use it after a 401 so that concurrent requests, which were
// all refused for the same token, cause one token request instead of one each.
func (c *CachedTokenSource) InvalidateToken(rejected string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cached != nil && c.cached.AccessToken == rejected {
		c.cached = nil
	}
}
