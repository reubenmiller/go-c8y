package microservice

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBasicAuthRequest(t *testing.T, user, password string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/devices", nil)
	r.SetBasicAuth(user, password)
	return r
}

// makeUnsignedJWT builds a syntactically valid (unsigned) JWT carrying the
// given claims. The signature is not validated by AuthFromRequest, only parsed.
func makeUnsignedJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func Test_AuthFromRequest_BasicAuth(t *testing.T) {
	r := newBasicAuthRequest(t, "t100/service_app", "secret")

	auth, ok := AuthFromRequest(r)
	require.True(t, ok)
	assert.Equal(t, "t100", auth.Tenant)
	assert.Equal(t, "service_app", auth.Username)
	assert.Equal(t, "secret", auth.Password)
}

func Test_AuthFromRequest_BasicAuthWithoutTenant(t *testing.T) {
	r := newBasicAuthRequest(t, "someuser", "secret")

	auth, ok := AuthFromRequest(r)
	require.True(t, ok)
	assert.Equal(t, "", auth.Tenant)
	assert.Equal(t, "someuser", auth.Username)
}

func Test_AuthFromRequest_BearerToken(t *testing.T) {
	token := makeUnsignedJWT(t, map[string]any{
		"ten": "t200",
		"sub": "user@example.com",
	})
	r := httptest.NewRequest(http.MethodGet, "/devices", nil)
	r.Header.Set("Authorization", "Bearer "+token)

	auth, ok := AuthFromRequest(r)
	require.True(t, ok)
	assert.Equal(t, "t200", auth.Tenant)
	assert.Equal(t, token, auth.Token)
	assert.Equal(t, "t200", TenantFromRequest(r))
}

func Test_AuthFromRequest_NoCredentials(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/devices", nil)

	_, ok := AuthFromRequest(r)
	assert.False(t, ok)
	assert.Equal(t, "", TenantFromRequest(r))
}

func Test_TenantContext_ForwardsCallerCredentials(t *testing.T) {
	ms := &Microservice{}

	var gotAuth authentication.AuthOptions
	var gotTenant string
	handler := ms.TenantContext()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, _ = authentication.AuthFromContext(r.Context())
		gotTenant = TenantFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, newBasicAuthRequest(t, "t100/jane", "janepass"))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "t100", gotTenant)
	assert.Equal(t, "t100", gotAuth.Tenant)
	assert.Equal(t, "jane", gotAuth.Username)
	assert.Equal(t, "janepass", gotAuth.Password)
}

func Test_TenantContext_UseServiceUser(t *testing.T) {
	ms := &Microservice{}
	ms.SetServiceUsers([]model.ServiceUser{
		{Tenant: "t100", Username: "service_app", Password: "servicepass"},
		{Tenant: "t200", Username: "service_app", Password: "otherpass"},
	})

	var gotAuth authentication.AuthOptions
	handler := ms.TenantContext(TenantContextOptions{UseServiceUser: true})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth, _ = authentication.AuthFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

	// The caller's credentials are only used to identify the tenant; downstream
	// calls use the tenant's service user
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, newBasicAuthRequest(t, "t200/jane", "janepass"))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "t200", gotAuth.Tenant)
	assert.Equal(t, "service_app", gotAuth.Username)
	assert.Equal(t, "otherpass", gotAuth.Password)
}

func Test_TenantContext_UnauthorizedWhenMissingCredentials(t *testing.T) {
	ms := &Microservice{}

	handler := ms.TenantContext()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/devices", nil))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "security/Unauthorized") //nolint:misspell
}

func Test_TenantContext_UnauthorizedWhenTenantNotSubscribed(t *testing.T) {
	ms := &Microservice{}
	ms.SetServiceUsers([]model.ServiceUser{
		{Tenant: "t100", Username: "service_app", Password: "servicepass"},
	})

	var gotErr error
	handler := ms.TenantContext(TenantContextOptions{
		UseServiceUser: true,
		OnError: func(w http.ResponseWriter, r *http.Request, err error) {
			gotErr = err
			w.WriteHeader(http.StatusForbidden)
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, newBasicAuthRequest(t, "t999/jane", "janepass"))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.ErrorIs(t, gotErr, ErrTenantNotSubscribed)
}

func Test_ForEachTenant(t *testing.T) {
	ms := &Microservice{}
	ms.SetServiceUsers([]model.ServiceUser{
		{Tenant: "t100", Username: "service_app", Password: "pass1"},
		{Tenant: "t200", Username: "service_app", Password: "pass2"},
	})

	var tenants []string
	err := ms.ForEachTenant(t.Context(), func(ctx context.Context, user model.ServiceUser) error {
		auth, ok := authentication.AuthFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, user.Tenant, auth.Tenant)
		tenants = append(tenants, user.Tenant)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"t100", "t200"}, tenants)
}
