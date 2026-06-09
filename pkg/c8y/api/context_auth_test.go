package api

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"sync"
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
