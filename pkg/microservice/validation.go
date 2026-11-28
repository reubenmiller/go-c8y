package microservice

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"golang.org/x/sync/singleflight"
)

// CredentialValidation selects whether the TenantContext middleware checks the
// caller's credentials with the platform.
type CredentialValidation int

const (
	// ValidateDefault validates (cached) when UseServiceUser is true, and does
	// not validate otherwise. In user scope the downstream platform call
	// already checks the forwarded credentials.
	ValidateDefault CredentialValidation = iota

	// ValidateNone trusts the request as forwarded by the platform proxy and
	// makes no extra calls.
	ValidateNone

	// ValidateCached checks each distinct credential once with
	// GET /user/currentUser and caches the result (see ValidationTTL).
	ValidateCached
)

// Default validation cache settings.
const (
	DefaultValidationTTL      = 5 * time.Minute
	validationNegativeTTL     = 10 * time.Second
	validationCacheMaxEntries = 10000
)

var (
	// ErrInvalidCredentials means the platform rejected the caller's
	// credentials. The default error handler answers 401.
	ErrInvalidCredentials = errors.New("microservice: the platform rejected the caller's credentials")

	// ErrValidationUnavailable means the credentials could not be checked
	// because the platform did not answer. The default error handler answers
	// 503. It is never cached.
	ErrValidationUnavailable = errors.New("microservice: could not validate credentials with the platform")

	// ErrMissingRole means the caller lacks a role required by RequireRoles.
	// The default error handler answers 403.
	ErrMissingRole = errors.New("microservice: caller is missing a required role")
)

// CallerInfo describes a caller whose credentials were validated.
type CallerInfo struct {
	Tenant   string
	UserName string
	Roles    []string
}

// HasRoles reports whether the caller has all given roles.
func (c CallerInfo) HasRoles(roles ...string) bool {
	for _, r := range roles {
		if !slices.Contains(c.Roles, r) {
			return false
		}
	}
	return true
}

type callerContextKey struct{}

// CallerFromContext returns the validated caller bound to the context by the
// TenantContext or RequireRoles middleware. The second value is false when
// the request was not validated.
func CallerFromContext(ctx context.Context) (CallerInfo, bool) {
	c, ok := ctx.Value(callerContextKey{}).(CallerInfo)
	return c, ok
}

// Validator checks credentials and describes the caller. It replaces the
// built-in GET /user/currentUser check (results are still cached).
type Validator func(ctx context.Context, auth authentication.AuthOptions) (CallerInfo, error)

func (o TenantContextOptions) validation() CredentialValidation {
	if o.Validation != ValidateDefault {
		return o.Validation
	}
	if o.UseServiceUser {
		return ValidateCached
	}
	return ValidateNone
}

func (o TenantContextOptions) ttl() time.Duration {
	if o.ValidationTTL > 0 {
		return o.ValidationTTL
	}
	return DefaultValidationTTL
}

// --- cache ---

type cacheEntry struct {
	key     string
	caller  CallerInfo
	err     error
	expires time.Time
}

// callerCache is an LRU cache of validation results keyed by a hash of the
// credential. Raw credentials are never stored.
type callerCache struct {
	mu      sync.Mutex
	entries map[string]*list.Element
	order   *list.List
	max     int
	group   singleflight.Group
}

func newCallerCache(max int) *callerCache {
	return &callerCache{entries: map[string]*list.Element{}, order: list.New(), max: max}
}

func (c *callerCache) get(key string) (*cacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	e := el.Value.(*cacheEntry)
	if time.Now().After(e.expires) {
		c.order.Remove(el)
		delete(c.entries, key)
		return nil, false
	}
	c.order.MoveToFront(el)
	return e, true
}

func (c *callerCache) put(e *cacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.entries[e.key]; ok {
		el.Value = e
		c.order.MoveToFront(el)
		return
	}
	c.entries[e.key] = c.order.PushFront(e)
	for c.order.Len() > c.max {
		oldest := c.order.Back()
		c.order.Remove(oldest)
		delete(c.entries, oldest.Value.(*cacheEntry).key)
	}
}

func (m *Microservice) callers() *callerCache {
	m.callerCacheOnce.Do(func() {
		m.callerCache = newCallerCache(validationCacheMaxEntries)
	})
	return m.callerCache
}

// credentialKey returns a hash identifying the credential of the request.
func credentialKey(r *http.Request, fromCookie bool) string {
	var raw string
	if fromCookie {
		if c, err := r.Cookie(CookieAuthorization); err == nil {
			raw = "cookie:" + c.Value
		}
	} else {
		raw = "header:" + r.Header.Get("Authorization")
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// validateCaller returns the validated caller for the request, using the cache.
func (m *Microservice) validateCaller(r *http.Request, auth authentication.AuthOptions, fromCookie bool, opts TenantContextOptions) (CallerInfo, error) {
	cache := m.callers()
	key := credentialKey(r, fromCookie)
	if e, ok := cache.get(key); ok {
		return e.caller, e.err
	}

	v, _, _ := cache.group.Do(key, func() (any, error) {
		if e, ok := cache.get(key); ok {
			return e, nil
		}
		var caller CallerInfo
		var err error
		if opts.Validator != nil {
			caller, err = opts.Validator(r.Context(), auth)
		} else {
			caller, err = m.currentUser(r, auth, fromCookie)
		}
		if err != nil && !errors.Is(err, ErrValidationUnavailable) && !errors.Is(err, ErrInvalidCredentials) {
			err = fmt.Errorf("%w: %v", ErrInvalidCredentials, err) //nolint:errorlint // the second error is described, not wrapped
		}
		if caller.Tenant == "" && err == nil {
			caller.Tenant = auth.Tenant
		}
		e := &cacheEntry{key: key, caller: caller, err: err}
		switch {
		case err == nil:
			e.expires = time.Now().Add(opts.ttl())
			if auth.Token != "" {
				if claims, perr := authentication.ParseToken(auth.Token); perr == nil && claims.ExpiresAt != nil && claims.ExpiresAt.Before(e.expires) {
					// The claim only shortens the TTL, so an unverified
					// value cannot extend access.
					e.expires = claims.ExpiresAt.Time
				}
			}
			cache.put(e)
		case errors.Is(err, ErrValidationUnavailable):
			// Never cache outages.
		default:
			e.expires = time.Now().Add(validationNegativeTTL)
			cache.put(e)
		}
		return e, nil
	})
	e := v.(*cacheEntry)
	return e.caller, e.err
}

// currentUser validates the request's credentials with GET /user/currentUser,
// forwarding the caller's Authorization header (or cookie and XSRF token).
func (m *Microservice) currentUser(r *http.Request, auth authentication.AuthOptions, fromCookie bool) (CallerInfo, error) {
	if m.Client == nil || m.Client.BaseURL() == nil {
		return CallerInfo{}, fmt.Errorf("%w: no platform client", ErrValidationUnavailable)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	u := strings.TrimSuffix(m.Client.BaseURL().String(), "/") + "/user/currentUser"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return CallerInfo{}, fmt.Errorf("%w: %v", ErrValidationUnavailable, err) //nolint:errorlint // the second error is described, not wrapped
	}
	req.Header.Set("Accept", "application/json")
	if fromCookie {
		if c, cerr := r.Cookie(CookieAuthorization); cerr == nil {
			req.AddCookie(c)
		}
		if xsrf := XSRFTokenFromRequest(r); xsrf != "" {
			req.Header.Set(HeaderXSRFToken, xsrf)
		}
	} else {
		req.Header.Set("Authorization", r.Header.Get("Authorization"))
	}

	resp, err := m.httpClientForValidation().Do(req)
	if err != nil {
		return CallerInfo{}, fmt.Errorf("%w: %v", ErrValidationUnavailable, err) //nolint:errorlint // the second error is described, not wrapped
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return CallerInfo{}, fmt.Errorf("%w (status %d)", ErrInvalidCredentials, resp.StatusCode)
	case resp.StatusCode >= 500:
		return CallerInfo{}, fmt.Errorf("%w (status %d)", ErrValidationUnavailable, resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return CallerInfo{}, fmt.Errorf("%w (status %d)", ErrInvalidCredentials, resp.StatusCode)
	}

	var user struct {
		UserName       string `json:"userName"`
		EffectiveRoles []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"effectiveRoles"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return CallerInfo{}, fmt.Errorf("%w: unreadable current user: %v", ErrValidationUnavailable, err) //nolint:errorlint // the second error is described, not wrapped
	}
	caller := CallerInfo{Tenant: auth.Tenant, UserName: user.UserName}
	for _, role := range user.EffectiveRoles {
		id := role.ID
		if id == "" {
			id = role.Name
		}
		caller.Roles = append(caller.Roles, id)
	}
	return caller, nil
}

// httpClientForValidation sends validation requests with the platform
// client's transport (TLS settings), but without its credentials.
func (m *Microservice) httpClientForValidation() *http.Client {
	if m.Client != nil && m.Client.Resty() != nil {
		return &http.Client{Transport: m.Client.Resty().Client().Transport}
	}
	return http.DefaultClient
}

// RequireRoles returns middleware that only lets callers with all given roles
// through. It validates the caller (cached, like ValidateCached) even when the
// surrounding TenantContext does not validate, because roles are only known
// after validation. Missing credentials answer 401, missing roles 403.
func (m *Microservice) RequireRoles(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			caller, ok := CallerFromContext(r.Context())
			if !ok {
				auth, fromCookie, found := authFromRequest(r)
				if !found {
					unauthorizedHandler(w, r, ErrMissingCredentials)
					return
				}
				var err error
				caller, err = m.validateCaller(r, auth, fromCookie, TenantContextOptions{})
				if err != nil {
					unauthorizedHandler(w, r, err)
					return
				}
				r = r.WithContext(context.WithValue(r.Context(), callerContextKey{}, caller))
			}
			if !caller.HasRoles(roles...) {
				unauthorizedHandler(w, r, fmt.Errorf("%w: needs %s", ErrMissingRole, strings.Join(roles, ", ")))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
