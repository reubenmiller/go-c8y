package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newShapeClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return NewClient(ClientOptions{
		BaseURL:          ts.URL,
		SkipTenantLookup: true,
		Auth: authentication.AuthOptions{
			Tenant: "t100", Username: "user", Password: "pass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
	})
}

type fixedResolver struct{ id string }

func (r fixedResolver) ResolveID(context.Context) (source.ResolveResult, error) {
	return source.ResolveResult{ID: r.id, Meta: map[string]any{"source": "custom"}}, nil
}
func (r fixedResolver) String() string { return "fixed:" + r.id }

// The deprecated flat Client fields and the nested accessors are the same
// service instance.
//nolint:staticcheck // checks the deprecated aliases on purpose
func TestClient_NestedServiceAliases(t *testing.T) {
	c := newShapeClient(t, func(http.ResponseWriter, *http.Request) {})

	assert.NotNil(t, c.Applications.Versions)
	assert.Same(t, c.ApplicationVersions, c.Applications.Versions)
	assert.NotNil(t, c.Tenants.DeviceStatistics)
	assert.Same(t, c.DeviceStatistics, c.Tenants.DeviceStatistics)
	assert.NotNil(t, c.Tenants.LoginTokens)
	assert.Same(t, c.LoginTokens, c.Tenants.LoginTokens)
	assert.NotNil(t, c.Users.DevicePermissions)
	assert.Same(t, c.DevicePermissions, c.Users.DevicePermissions)
	assert.NotNil(t, c.UserRoles.Inventory)
	assert.Same(t, c.InventoryRoles, c.UserRoles.Inventory)

	assert.NotNil(t, c.Users.Current)
	assert.Same(t, c.Users.CurrentUser, c.Users.Current)
	assert.NotNil(t, c.Microservices.Current)
	assert.Same(t, &c.Microservices.CurrentMicroservice, c.Microservices.Current)
}

// A resolver registered on Client.ManagedObjects applies to Devices.
func TestClient_DevicesShareManagedObjectsResolver(t *testing.T) {
	var mu sync.Mutex
	var paths []string
	c := newShapeClient(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"4242"}`))
	})
	c.ManagedObjects.RegisterResolver("custom", fixedResolver{id: "4242"})

	res := c.Devices.Get(context.Background(), "custom:anything", managedobjects.GetOptions{})
	require.NoError(t, res.Err)
	assert.Equal(t, "4242", res.Data.ID())
	require.Len(t, paths, 1)
	assert.Equal(t, "/inventory/managedObjects/4242", paths[0])
}
