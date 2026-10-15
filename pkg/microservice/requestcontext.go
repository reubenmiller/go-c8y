package microservice

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
)

// Errors returned by the tenant-context middleware. They are passed to the
// TenantContextOptions.OnError handler so callers can distinguish between an
// unauthenticated request and an unsubscribed tenant.
var (
	// ErrMissingCredentials indicates that the incoming request did not contain
	// a usable Authorization header.
	ErrMissingCredentials = errors.New("microservice: request has no Cumulocity credentials")

	// ErrTenantNotSubscribed indicates that no service user is available for
	// the tenant of the incoming request.
	ErrTenantNotSubscribed = errors.New("microservice: no service user found for tenant")

	// ErrInvalidXSRFToken indicates that a state-changing request was
	// authenticated with the "authorization" cookie but the X-XSRF-TOKEN header
	// was missing or did not match the XSRF-TOKEN cookie.
	ErrInvalidXSRFToken = errors.New("microservice: missing or invalid XSRF token for cookie-authenticated request")
)

const (
	// CookieAuthorization is the cookie holding the platform JWT (OAI-Secure).
	CookieAuthorization = "authorization"
	// CookieXSRFToken is the cookie holding the XSRF token (OAI-Secure).
	CookieXSRFToken = "XSRF-TOKEN"
	// HeaderXSRFToken is the request header echoing the XSRF token.
	HeaderXSRFToken = "X-XSRF-TOKEN"
)

type xsrfContextKey struct{}

// XSRFTokenFromRequest returns the XSRF token sent by the caller in the
// X-XSRF-TOKEN header, or an empty string when absent. Cookie-authenticated
// calls made on behalf of the user may need to forward it (the go-c8y client
// sends it as the X-XSRF-TOKEN header for OAI-Secure sessions).
func XSRFTokenFromRequest(r *http.Request) string {
	return r.Header.Get(HeaderXSRFToken)
}

// XSRFTokenFromContext returns the XSRF token bound to the context by the
// TenantContext middleware for cookie-authenticated requests, or an empty
// string.
func XSRFTokenFromContext(ctx context.Context) string {
	v, _ := ctx.Value(xsrfContextKey{}).(string)
	return v
}

// AuthFromRequest extracts the Cumulocity credentials from an incoming HTTP
// request. When the Cumulocity platform proxies a request to a microservice
// (via /service/{name}/...), the Authorization header carries either:
//
//   - Basic auth with the username in "{tenant}/{username}" format, or
//   - a Bearer token (JWT) whose "ten" claim identifies the tenant.
//
// Requests from Cumulocity web applications (e.g. Device Management) carry no
// Authorization header; instead they send the OAI-Secure session cookies. As a
// fallback, the platform JWT in the "authorization" cookie is used as a token
// (the tenant is taken from its "ten" claim). Basic and Bearer headers always
// take precedence over the cookie.
//
// The returned AuthOptions can be attached to a context via api.WithAuth so
// that downstream API calls execute on behalf of the caller. The second return
// value is false when the request carries no usable credentials.
func AuthFromRequest(r *http.Request) (authentication.AuthOptions, bool) {
	opts, _, ok := authFromRequest(r)
	return opts, ok
}

// authFromRequest is like AuthFromRequest but also reports whether the
// credentials were taken from the authorization cookie.
func authFromRequest(r *http.Request) (opts authentication.AuthOptions, fromCookie, ok bool) {
	if username, password, ok := r.BasicAuth(); ok {
		tenant, user := authentication.SplitTenantUser(username)
		return authentication.AuthOptions{
			Tenant:   tenant,
			Username: user,
			Password: password,
		}, false, true
	}

	header := r.Header.Get("Authorization")
	if token, ok := strings.CutPrefix(header, "Bearer "); ok && token != "" {
		return tokenAuth(token), false, true
	}

	if c, err := r.Cookie(CookieAuthorization); err == nil && c.Value != "" {
		return tokenAuth(c.Value), true, true
	}

	return authentication.AuthOptions{}, false, false
}

func tokenAuth(token string) authentication.AuthOptions {
	opts := authentication.AuthOptions{Token: token}
	if claim, err := authentication.ParseToken(token); err == nil {
		opts.Tenant = claim.Tenant
	}
	return opts
}

// validXSRF reports whether the X-XSRF-TOKEN header matches the XSRF-TOKEN
// cookie (constant-time comparison).
func validXSRF(r *http.Request) bool {
	header := r.Header.Get(HeaderXSRFToken)
	c, err := r.Cookie(CookieXSRFToken)
	if header == "" || err != nil || c.Value == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(header), []byte(c.Value)) == 1
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	}
	return false
}

// TenantFromRequest returns the tenant ID of the caller of an incoming HTTP
// request, or an empty string when the request carries no usable credentials.
func TenantFromRequest(r *http.Request) string {
	if auth, ok := AuthFromRequest(r); ok {
		return auth.Tenant
	}
	return ""
}

// TenantFromContext returns the tenant bound to the context by the
// TenantContext middleware (or by api.WithAuth / api.WithServiceUser), or an
// empty string when the context carries no tenant-scoped credentials.
func TenantFromContext(ctx context.Context) string {
	if auth, ok := authentication.AuthFromContext(ctx); ok {
		return auth.Tenant
	}
	return ""
}

// TenantContextOptions configures the TenantContext middleware.
type TenantContextOptions struct {
	// UseServiceUser controls which credentials are bound to the request
	// context for downstream API calls:
	//
	//   false (default): the caller's own credentials are forwarded, so
	//   downstream calls run with the caller's permissions (user scope).
	//
	//   true: the caller's credentials are only used to identify the tenant;
	//   downstream calls run as the tenant's service user (tenant scope, with
	//   the roles requested in the microservice manifest).
	UseServiceUser bool

	// DisableXSRFCheck turns off the CSRF check for cookie-authenticated
	// requests. By default, requests authenticated only via the "authorization"
	// cookie (browser sessions) that use a state-changing method (anything but
	// GET, HEAD, OPTIONS, TRACE) must send an X-XSRF-TOKEN header equal to the
	// XSRF-TOKEN cookie, otherwise OnError is called with ErrInvalidXSRFToken
	// and, by default, a 403 is returned. Requests authenticated with an
	// Authorization header are never checked, as browsers do not add that
	// header automatically.
	DisableXSRFCheck bool

	// OnError is called when the request carries no usable credentials
	// (ErrMissingCredentials), when the XSRF check fails (ErrInvalidXSRFToken)
	// or, with UseServiceUser, when the tenant has no
	// service user (ErrTenantNotSubscribed). When nil, a 401 JSON response is
	// written.
	OnError func(w http.ResponseWriter, r *http.Request, err error)
}

// TenantContext returns a standard net/http middleware that binds the caller's
// tenant credentials to the request context, so that any m.Client API call made
// with r.Context() automatically executes on behalf of the caller's tenant.
// This mirrors the Cumulocity Java SDK behaviour where the per-request security
// context determines the tenant used for all downstream platform calls.
//
// The middleware is framework-agnostic: it works with net/http, chi, gorilla
// and, via adapters such as echo.WrapMiddleware, with most other routers.
//
//	mux := http.NewServeMux()
//	mux.Handle("/devices", ms.TenantContext()(http.HandlerFunc(listDevices)))
//
//	func listDevices(w http.ResponseWriter, r *http.Request) {
//		// Runs as the caller's tenant
//		result := ms.Client.Devices.List(r.Context(), devices.ListOptions{})
//		...
//	}
func (m *Microservice) TenantContext(opts ...TenantContextOptions) func(http.Handler) http.Handler {
	options := TenantContextOptions{}
	if len(opts) > 0 {
		options = opts[0]
	}
	onError := options.OnError
	if onError == nil {
		onError = unauthorizedHandler
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth, fromCookie, ok := authFromRequest(r)
			if !ok {
				onError(w, r, ErrMissingCredentials)
				return
			}

			if fromCookie && !options.DisableXSRFCheck && !isSafeMethod(r.Method) && !validXSRF(r) {
				onError(w, r, ErrInvalidXSRFToken)
				return
			}

			if options.UseServiceUser {
				user, found := m.GetServiceUser(auth.Tenant)
				if !found {
					onError(w, r, fmt.Errorf("%w: %s", ErrTenantNotSubscribed, auth.Tenant))
					return
				}
				auth = authentication.AuthOptions{
					Tenant:   user.Tenant,
					Username: user.Username,
					Password: user.Password,
				}
			}

			ctx := authentication.WithAuth(r.Context(), auth)
			if fromCookie {
				if xsrf := XSRFTokenFromRequest(r); xsrf != "" {
					ctx = context.WithValue(ctx, xsrfContextKey{}, xsrf)
				}
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func unauthorizedHandler(w http.ResponseWriter, _ *http.Request, err error) {
	w.Header().Set("Content-Type", "application/json")
	if errors.Is(err, ErrInvalidXSRFToken) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, `{"error":"security/Forbidden","message":%q}`, err.Error()) //nolint:misspell
		return
	}
	w.WriteHeader(http.StatusUnauthorized)
	// The error type string matches the spelling used by the Cumulocity platform
	fmt.Fprintf(w, `{"error":"security/Unauthorized","message":%q}`, err.Error()) //nolint:misspell
}
