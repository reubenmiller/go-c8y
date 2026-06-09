package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/devices"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/usergroups"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingServer captures the Authorization header and path of every request
// to /inventory and /user endpoints.
type recordingServer struct {
	mu       sync.Mutex
	requests []recordedRequest
	server   *httptest.Server
}

type recordedRequest struct {
	Path          string
	Authorization string
}

func newRecordingServer() *recordingServer {
	rs := &recordingServer{}
	rs.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rs.mu.Lock()
		rs.requests = append(rs.requests, recordedRequest{
			Path:          r.URL.Path,
			Authorization: r.Header.Get("Authorization"),
		})
		rs.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/inventory/managedObjects":
			_, _ = w.Write([]byte(`{"managedObjects":[],"statistics":{"currentPage":1,"pageSize":5}}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	return rs
}

func (rs *recordingServer) last(t *testing.T) recordedRequest {
	t.Helper()
	rs.mu.Lock()
	defer rs.mu.Unlock()
	require.NotEmpty(t, rs.requests)
	return rs.requests[len(rs.requests)-1]
}

func basicAuth(user, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
}

func Test_ContextAuth_OverridesClientCredentials(t *testing.T) {
	rs := newRecordingServer()
	defer rs.server.Close()

	client := NewClient(ClientOptions{
		BaseURL: rs.server.URL,
		Auth: authentication.AuthOptions{
			Tenant:   "t100",
			Username: "bootstrap",
			Password: "bootpass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
	})

	// Without a context override the bootstrap credentials are used
	result := client.Devices.List(context.Background(), devices.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, basicAuth("t100/bootstrap", "bootpass"), rs.last(t).Authorization)

	// With a service user bound to the context, the request runs as that tenant
	ctx := WithServiceUser(context.Background(), model.ServiceUser{
		Tenant:   "t200",
		Username: "service_app",
		Password: "servicepass",
	})
	result = client.Devices.List(ctx, devices.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, basicAuth("t200/service_app", "servicepass"), rs.last(t).Authorization)

	// Subsequent calls without the override fall back to the bootstrap credentials
	result = client.Devices.List(context.Background(), devices.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, basicAuth("t100/bootstrap", "bootpass"), rs.last(t).Authorization)
}

func Test_ContextAuth_TokenOverride(t *testing.T) {
	rs := newRecordingServer()
	defer rs.server.Close()

	client := NewClient(ClientOptions{
		BaseURL: rs.server.URL,
		Auth: authentication.AuthOptions{
			Tenant:   "t100",
			Username: "bootstrap",
			Password: "bootpass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
	})

	ctx := WithAuth(context.Background(), authentication.AuthOptions{
		Tenant: "t300",
		Token:  "my-token",
	})
	result := client.Devices.List(ctx, devices.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, "Bearer my-token", rs.last(t).Authorization)
}

func Test_ContextAuth_SetsTenantPathParam(t *testing.T) {
	rs := newRecordingServer()
	defer rs.server.Close()

	client := NewClient(ClientOptions{
		BaseURL: rs.server.URL,
		Auth: authentication.AuthOptions{
			Tenant:   "t100",
			Username: "bootstrap",
			Password: "bootpass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
	})

	// The {tenantId} path parameter defaults to the client-level tenant
	result := client.UserGroups.List(context.Background(), usergroups.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, "/user/t100/groups", rs.last(t).Path)

	// A service user in the context overrides the {tenantId} path parameter
	ctx := WithServiceUser(context.Background(), model.ServiceUser{
		Tenant:   "t200",
		Username: "service_app",
		Password: "servicepass",
	})
	result = client.UserGroups.List(ctx, usergroups.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, "/user/t200/groups", rs.last(t).Path)

	// WithTenant overrides only the path parameter, not the credentials
	ctx = WithTenant(context.Background(), "t400")
	result = client.UserGroups.List(ctx, usergroups.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, "/user/t400/groups", rs.last(t).Path)
	assert.Equal(t, basicAuth("t100/bootstrap", "bootpass"), rs.last(t).Authorization)
}

// tokenExchangeServer extends the recording server with an OAI-Secure token
// endpoint that issues sequentially numbered tokens. Only tenant-scoped token
// requests (tenant_id query parameter set) are counted and served — these are
// the ones produced by the context token exchange. The client's own internal
// token source (no tenant_id) always receives a 404 so it falls back to the
// client-level basic credentials.
func newTokenExchangeServer(tokenStatus int) (*recordingServer, *int64) {
	rs := &recordingServer{}
	tokenRequests := new(int64)
	rs.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tenant/oauth/token" {
			if r.URL.Query().Get("tenant_id") == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			n := atomic.AddInt64(tokenRequests, 1)
			if tokenStatus >= 400 {
				w.WriteHeader(tokenStatus)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"access_token":"exchanged-token-%d"}`, n)
			return
		}

		rs.mu.Lock()
		rs.requests = append(rs.requests, recordedRequest{
			Path:          r.URL.Path,
			Authorization: r.Header.Get("Authorization"),
		})
		rs.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"managedObjects":[],"statistics":{"currentPage":1,"pageSize":5}}`))
	}))
	return rs, tokenRequests
}

func Test_ContextTokenExchange_ExchangesAndCaches(t *testing.T) {
	rs, tokenRequests := newTokenExchangeServer(http.StatusOK)
	defer rs.server.Close()

	client := NewClient(ClientOptions{
		BaseURL: rs.server.URL,
		Auth: authentication.AuthOptions{
			Tenant:   "t100",
			Username: "bootstrap",
			Password: "bootpass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
		ContextTokenExchange: true,
	})

	ctx := WithServiceUser(context.Background(), model.ServiceUser{
		Tenant:   "t200",
		Username: "service_app",
		Password: "servicepass",
	})

	// First call pays one login round-trip and uses the exchanged token
	result := client.Devices.List(ctx, devices.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, "Bearer exchanged-token-1", rs.last(t).Authorization)
	assert.EqualValues(t, 1, atomic.LoadInt64(tokenRequests))

	// Second call reuses the cached token (no extra login)
	result = client.Devices.List(ctx, devices.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, "Bearer exchanged-token-1", rs.last(t).Authorization)
	assert.EqualValues(t, 1, atomic.LoadInt64(tokenRequests))

	// A different tenant gets its own token
	ctx2 := WithServiceUser(context.Background(), model.ServiceUser{
		Tenant:   "t300",
		Username: "service_app",
		Password: "otherpass",
	})
	result = client.Devices.List(ctx2, devices.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, "Bearer exchanged-token-2", rs.last(t).Authorization)
	assert.EqualValues(t, 2, atomic.LoadInt64(tokenRequests))

	// Requests without context credentials are unaffected
	result = client.Devices.List(context.Background(), devices.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, basicAuth("t100/bootstrap", "bootpass"), rs.last(t).Authorization)
	assert.EqualValues(t, 2, atomic.LoadInt64(tokenRequests))
}

func Test_ContextTokenExchange_FallsBackToBasicWithCooldown(t *testing.T) {
	rs, tokenRequests := newTokenExchangeServer(http.StatusNotFound)
	defer rs.server.Close()

	client := NewClient(ClientOptions{
		BaseURL: rs.server.URL,
		Auth: authentication.AuthOptions{
			Tenant:   "t100",
			Username: "bootstrap",
			Password: "bootpass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
		ContextTokenExchange: true,
	})

	ctx := WithServiceUser(context.Background(), model.ServiceUser{
		Tenant:   "t200",
		Username: "service_app",
		Password: "servicepass",
	})

	// Exchange fails -> request proceeds with basic auth
	result := client.Devices.List(ctx, devices.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, basicAuth("t200/service_app", "servicepass"), rs.last(t).Authorization)
	exchangeAttempts := atomic.LoadInt64(tokenRequests)
	assert.GreaterOrEqual(t, exchangeAttempts, int64(1))

	// The failed exchange is in cooldown -> no further login attempts
	result = client.Devices.List(ctx, devices.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, basicAuth("t200/service_app", "servicepass"), rs.last(t).Authorization)
	assert.Equal(t, exchangeAttempts, atomic.LoadInt64(tokenRequests))
}

func Test_ContextTokenExchange_Disabled(t *testing.T) {
	rs, tokenRequests := newTokenExchangeServer(http.StatusOK)
	defer rs.server.Close()

	client := NewClient(ClientOptions{
		BaseURL: rs.server.URL,
		Auth: authentication.AuthOptions{
			Tenant:   "t100",
			Username: "bootstrap",
			Password: "bootpass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
	})

	ctx := WithServiceUser(context.Background(), model.ServiceUser{
		Tenant:   "t200",
		Username: "service_app",
		Password: "servicepass",
	})

	result := client.Devices.List(ctx, devices.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, basicAuth("t200/service_app", "servicepass"), rs.last(t).Authorization)
	assert.EqualValues(t, 0, atomic.LoadInt64(tokenRequests))
}

func Test_ContextTokenExchange_RetriesWithBasicAfter401(t *testing.T) {
	// Server rejects all bearer tokens (e.g. token invalidated server-side) but
	// accepts basic credentials.
	rs := &recordingServer{}
	tokenRequests := new(int64)
	rs.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tenant/oauth/token" {
			n := atomic.AddInt64(tokenRequests, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"access_token":"exchanged-token-%d"}`, n)
			return
		}

		rs.mu.Lock()
		rs.requests = append(rs.requests, recordedRequest{
			Path:          r.URL.Path,
			Authorization: r.Header.Get("Authorization"),
		})
		rs.mu.Unlock()

		if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"security/Unauthorized","message":"invalid token"}`)) //nolint:misspell
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"managedObjects":[],"statistics":{"currentPage":1,"pageSize":5}}`))
	}))
	defer rs.server.Close()

	client := NewClient(ClientOptions{
		BaseURL: rs.server.URL,
		Auth: authentication.AuthOptions{
			Tenant:   "t100",
			Username: "bootstrap",
			Password: "bootpass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
		ContextTokenExchange: true,
	})

	ctx := WithServiceUser(context.Background(), model.ServiceUser{
		Tenant:   "t200",
		Username: "service_app",
		Password: "servicepass",
	})

	result := client.Devices.List(ctx, devices.ListOptions{})
	require.NoError(t, result.Err)

	// First attempt used the exchanged token (rejected), the retry used basic auth
	rs.mu.Lock()
	var inventory []recordedRequest
	for _, req := range rs.requests {
		if req.Path == "/inventory/managedObjects" {
			inventory = append(inventory, req)
		}
	}
	rs.mu.Unlock()
	require.Len(t, inventory, 2)
	assert.True(t, strings.HasPrefix(inventory[0].Authorization, "Bearer exchanged-token-"), "first attempt should use the exchanged token, got %q", inventory[0].Authorization)
	assert.Equal(t, basicAuth("t200/service_app", "servicepass"), inventory[1].Authorization)
}
