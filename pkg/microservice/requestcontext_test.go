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

func newCookieRequest(t *testing.T, method, token, xsrfCookie, xsrfHeader string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(method, "/devices", nil)
	if token != "" {
		r.AddCookie(&http.Cookie{Name: CookieAuthorization, Value: token})
	}
	if xsrfCookie != "" {
		r.AddCookie(&http.Cookie{Name: CookieXSRFToken, Value: xsrfCookie})
	}
	if xsrfHeader != "" {
		r.Header.Set(HeaderXSRFToken, xsrfHeader)
	}
	return r
}

func Test_AuthFromRequest_CookieOnly(t *testing.T) {
	token := makeUnsignedJWT(t, map[string]any{"ten": "t300", "sub": "jane"})
	r := newCookieRequest(t, http.MethodGet, token, "x1", "x1")

	auth, ok := AuthFromRequest(r)
	require.True(t, ok)
	assert.Equal(t, "t300", auth.Tenant)
	assert.Equal(t, token, auth.Token)
	assert.Equal(t, "t300", TenantFromRequest(r))
	assert.Equal(t, "x1", XSRFTokenFromRequest(r))
}

func Test_AuthFromRequest_HeaderBeatsCookie(t *testing.T) {
	cookieToken := makeUnsignedJWT(t, map[string]any{"ten": "cookie"})
	headerToken := makeUnsignedJWT(t, map[string]any{"ten": "header"})

	r := newCookieRequest(t, http.MethodGet, cookieToken, "", "")
	r.Header.Set("Authorization", "Bearer "+headerToken)
	auth, ok := AuthFromRequest(r)
	require.True(t, ok)
	assert.Equal(t, "header", auth.Tenant)

	r = newCookieRequest(t, http.MethodGet, cookieToken, "", "")
	r.SetBasicAuth("t100/jane", "pw")
	auth, ok = AuthFromRequest(r)
	require.True(t, ok)
	assert.Equal(t, "t100", auth.Tenant)
	assert.Equal(t, "pw", auth.Password)
}

func Test_TenantContext_CookieXSRF(t *testing.T) {
	token := makeUnsignedJWT(t, map[string]any{"ten": "t300"})
	ms := &Microservice{}

	run := func(opts TenantContextOptions, r *http.Request) (int, string, string) {
		var tenant, xsrf string
		h := ms.TenantContext(opts)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenant = TenantFromContext(r.Context())
			xsrf = XSRFTokenFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code, tenant, xsrf
	}

	t.Run("GET without XSRF allowed", func(t *testing.T) {
		code, tenant, _ := run(TenantContextOptions{}, newCookieRequest(t, http.MethodGet, token, "", ""))
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, "t300", tenant)
	})
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(m+" matching XSRF passes", func(t *testing.T) {
			code, tenant, xsrf := run(TenantContextOptions{}, newCookieRequest(t, m, token, "abc", "abc"))
			assert.Equal(t, http.StatusOK, code)
			assert.Equal(t, "t300", tenant)
			assert.Equal(t, "abc", xsrf)
		})
		t.Run(m+" missing XSRF header forbidden", func(t *testing.T) {
			code, _, _ := run(TenantContextOptions{}, newCookieRequest(t, m, token, "abc", ""))
			assert.Equal(t, http.StatusForbidden, code)
		})
	}
	t.Run("mismatch forbidden", func(t *testing.T) {
		code, _, _ := run(TenantContextOptions{}, newCookieRequest(t, http.MethodPost, token, "abc", "abd"))
		assert.Equal(t, http.StatusForbidden, code)
	})
	t.Run("missing cookie forbidden", func(t *testing.T) {
		code, _, _ := run(TenantContextOptions{}, newCookieRequest(t, http.MethodPost, token, "", "abc"))
		assert.Equal(t, http.StatusForbidden, code)
	})
	t.Run("check disabled", func(t *testing.T) {
		code, _, _ := run(TenantContextOptions{DisableXSRFCheck: true}, newCookieRequest(t, http.MethodPost, token, "", ""))
		assert.Equal(t, http.StatusOK, code)
	})
	t.Run("header auth is not checked", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/devices", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		code, _, _ := run(TenantContextOptions{}, r)
		assert.Equal(t, http.StatusOK, code)
	})
	t.Run("custom OnError receives ErrInvalidXSRFToken", func(t *testing.T) {
		var got error
		code, _, _ := run(TenantContextOptions{OnError: func(w http.ResponseWriter, _ *http.Request, err error) {
			got = err
			w.WriteHeader(http.StatusTeapot)
		}}, newCookieRequest(t, http.MethodPost, token, "", ""))
		assert.Equal(t, http.StatusTeapot, code)
		assert.ErrorIs(t, got, ErrInvalidXSRFToken)
	})
}
