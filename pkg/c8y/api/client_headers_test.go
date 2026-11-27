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
	return NewClient(ClientOptions{
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
	assert.NotEmpty(t, seen.Get("User-Agent"))
	assert.NotEmpty(t, seen.Get("X-Application"))
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
