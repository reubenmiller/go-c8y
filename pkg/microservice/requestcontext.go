package microservice

import (
	"context"
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
)

// AuthFromRequest extracts the Cumulocity credentials from an incoming HTTP
// request. When the Cumulocity platform proxies a request to a microservice
// (via /service/{name}/...), the Authorization header carries either:
//
//   - Basic auth with the username in "{tenant}/{username}" format, or
//   - a Bearer token (JWT) whose "ten" claim identifies the tenant.
//
// The returned AuthOptions can be attached to a context via api.WithAuth so
// that downstream API calls execute on behalf of the caller. The second return
// value is false when the request carries no usable credentials.
func AuthFromRequest(r *http.Request) (authentication.AuthOptions, bool) {
	if username, password, ok := r.BasicAuth(); ok {
		tenant, user := authentication.SplitTenantUser(username)
		return authentication.AuthOptions{
			Tenant:   tenant,
			Username: user,
			Password: password,
		}, true
	}

	header := r.Header.Get("Authorization")
	if token, ok := strings.CutPrefix(header, "Bearer "); ok && token != "" {
		opts := authentication.AuthOptions{Token: token}
		if claim, err := authentication.ParseToken(token); err == nil {
			opts.Tenant = claim.Tenant
		}
		return opts, true
	}

	return authentication.AuthOptions{}, false
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

	// OnError is called when the request carries no usable credentials
	// (ErrMissingCredentials) or, with UseServiceUser, when the tenant has no
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
			auth, ok := AuthFromRequest(r)
			if !ok {
				onError(w, r, ErrMissingCredentials)
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
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func unauthorizedHandler(w http.ResponseWriter, _ *http.Request, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	// The error type string matches the spelling used by the Cumulocity platform
	fmt.Fprintf(w, `{"error":"security/Unauthorized","message":%q}`, err.Error()) //nolint:misspell
}
