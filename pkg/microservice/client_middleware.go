package microservice

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
)

// HeaderApplicationKey identifies requests sent by an application.
const HeaderApplicationKey = "X-Cumulocity-Application-Key"

// serviceUserRefreshInterval limits refreshes triggered by 401 responses.
const serviceUserRefreshInterval = 10 * time.Second

// installClientMiddleware adds the application key header (when APPLICATION_KEY
// is set) and refreshes service users when a service-user request gets 401.
func (m *Microservice) installClientMiddleware() {
	if m.Client == nil {
		return
	}
	if key := os.Getenv("APPLICATION_KEY"); key != "" {
		m.Client.Use(func(next http.RoundTripper) http.RoundTripper {
			return api.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				r.Header.Set(HeaderApplicationKey, key)
				return next.RoundTrip(r)
			})
		})
	}
	m.Client.Use(func(next http.RoundTripper) http.RoundTripper {
		return api.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			resp, err := next.RoundTrip(r)
			if err == nil && resp.StatusCode == http.StatusUnauthorized {
				if auth, ok := authentication.AuthFromContext(r.Context()); ok && m.isServiceUser(auth) {
					m.refreshAfterUnauthorized()
				}
			}
			return resp, err
		})
	})
}

// refreshAfterUnauthorized reloads the service users in the background (at
// most once per serviceUserRefreshInterval), e.g. after a tenant's service
// user was recreated.
func (m *Microservice) refreshAfterUnauthorized() {
	now := time.Now().UnixNano()
	last := m.lastRefresh.Load()
	if now-last < int64(serviceUserRefreshInterval) || !m.lastRefresh.CompareAndSwap(last, now) {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := m.RefreshServiceUsers(ctx); err != nil {
			slog.Warn("Failed to refresh service users after 401", "err", err)
		}
	}()
}
