package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/stretchr/testify/assert"
)

func identityHeadersClient(t *testing.T, seen *http.Header) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seen = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"managedObjects":[]}`))
	}))
	t.Cleanup(srv.Close)
	return mustNewClient(ClientOptions{
		BaseURL:          srv.URL,
		SkipTenantLookup: true,
		Agent:            "my-agent",
		Auth:             authentication.AuthOptions{Username: "u", Password: "p", Tenant: "t", AuthType: []authentication.AuthType{authentication.AuthTypeBasic}},
	})
}

// The client identifies itself with User-Agent and X-Application, unless the
// caller set these headers.
func TestClient_IdentityHeadersDefaults(t *testing.T) {
	var seen http.Header
	client := identityHeadersClient(t, &seen)
	_ = client.ManagedObjects.List(context.Background(), managedobjects.ListOptions{})
	// ClientOptions.Agent is the User-Agent (not the resty default "go-resty/...")
	assert.Equal(t, "my-agent", seen.Get("User-Agent"))
	assert.Equal(t, "go-client", seen.Get("X-Application"))
	assert.Equal(t, "my-agent", client.UserAgent())
}

// Without ClientOptions.Agent the User-Agent is "go-client-v2". The websocket
// upgrade request sends the same identity headers as a REST request.
func TestClient_IdentityHeadersDefaultAgent(t *testing.T) {
	var seen http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"managedObjects":[]}`))
	}))
	t.Cleanup(srv.Close)
	client := mustNewClient(ClientOptions{BaseURL: srv.URL, SkipTenantLookup: true})
	_ = client.ManagedObjects.List(context.Background(), managedobjects.ListOptions{})
	assert.Equal(t, "go-client-v2", seen.Get("User-Agent"))
	assert.Equal(t, "go-client", seen.Get("X-Application"))

	wsHeader, err := client.WebsocketHeader(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, seen.Get("User-Agent"), wsHeader.Get("User-Agent"))
	assert.Equal(t, seen.Get("X-Application"), wsHeader.Get("X-Application"))
}

func TestClient_IdentityHeadersCanBeOverridden(t *testing.T) {
	var seen http.Header
	client := identityHeadersClient(t, &seen)
	ctx := WithExtraHeaders(context.Background(), map[string]string{
		"User-Agent":    "custom-agent",
		"X-Application": "custom-app",
		"X-Other":       "other",
	})
	_ = client.ManagedObjects.List(ctx, managedobjects.ListOptions{})
	assert.Equal(t, "custom-agent", seen.Get("User-Agent"))
	assert.Equal(t, "custom-app", seen.Get("X-Application"))
	assert.Equal(t, "other", seen.Get("X-Other"))
}

func TestClient_IdentityHeadersOnlyOverriddenHeader(t *testing.T) {
	var seen http.Header
	client := identityHeadersClient(t, &seen)
	_ = client.ManagedObjects.List(context.Background(), managedobjects.ListOptions{})
	defaultUserAgent := seen.Get("User-Agent")
	ctx := WithExtraHeaders(context.Background(), map[string]string{"X-Application": "custom-app"})
	_ = client.ManagedObjects.List(ctx, managedobjects.ListOptions{})
	assert.Equal(t, defaultUserAgent, seen.Get("User-Agent"))
	assert.Equal(t, "custom-app", seen.Get("X-Application"))
}
