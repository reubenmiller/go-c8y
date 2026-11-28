package microservice

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
	"time"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testclient"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePlatform answers GET /user/currentUser and counts the calls. Valid
// credentials are "<tenant>/<user>:good"; status, when set, overrides.
type fakePlatform struct {
	calls  atomic.Int32
	status atomic.Int32
	delay  time.Duration
	server *httptest.Server
}

func newFakePlatform(t *testing.T) *fakePlatform {
	t.Helper()
	fp := &fakePlatform{}
	fp.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/currentUser" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fp.calls.Add(1)
		time.Sleep(fp.delay)
		if s := fp.status.Load(); s != 0 {
			w.WriteHeader(int(s))
			return
		}
		authz := r.Header.Get("Authorization")
		if strings.HasPrefix(authz, "Bearer ") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"userName":"tokenuser","effectiveRoles":[{"id":"ROLE_INVENTORY_READ"}]}`))
			return
		}
		user, pass, ok := r.BasicAuth()
		if !ok || pass != "good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		roles := `[{"id":"ROLE_INVENTORY_READ"}]`
		if strings.HasSuffix(user, "/admin") {
			roles = `[{"id":"ROLE_INVENTORY_READ"},{"id":"ROLE_INVENTORY_ADMIN"}]`
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"userName":%q,"effectiveRoles":%s}`, user, roles)
	}))
	t.Cleanup(fp.server.Close)
	return fp
}

func newValidatingMicroservice(fp *fakePlatform) *Microservice {
	ms := &Microservice{
		Client: testclient.Must(api.ClientOptions{
			BaseURL:          fp.server.URL,
			SkipTenantLookup: true,
			Auth: authentication.AuthOptions{Tenant: "management", Username: "bootstrap", Password: "x",
				AuthType: []authentication.AuthType{authentication.AuthTypeBasic}},
		}),
	}
	ms.SetServiceUsers([]model.ServiceUser{
		{Tenant: "t123", Username: "service_app", Password: "servicepass"},
	})
	return ms
}

func basicReq(user, pass string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/devices", nil)
	r.SetBasicAuth(user, pass)
	return r
}

func serve(h http.Handler, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func Test_Validation_ServiceUserModeRejectsUnverifiedCaller(t *testing.T) {
	fp := newFakePlatform(t)
	ms := newValidatingMicroservice(fp)
	called := false
	h := ms.TenantContext(TenantContextOptions{UseServiceUser: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	w := serve(h, basicReq("t123/x", "wrong"))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, called, "the handler must not run")
	assert.EqualValues(t, 1, fp.calls.Load())
}

func Test_Validation_OptOut(t *testing.T) {
	fp := newFakePlatform(t)
	ms := newValidatingMicroservice(fp)
	var got authentication.AuthOptions
	h := ms.TenantContext(TenantContextOptions{UseServiceUser: true, Validation: ValidateNone})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = authentication.AuthFromContext(r.Context())
	}))

	w := serve(h, basicReq("t123/x", "wrong"))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "service_app", got.Username)
	assert.EqualValues(t, 0, fp.calls.Load())
}

func Test_Validation_UserScopeNotValidatedByDefault(t *testing.T) {
	fp := newFakePlatform(t)
	ms := newValidatingMicroservice(fp)
	h := ms.TenantContext()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	w := serve(h, basicReq("t123/jane", "anything"))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.EqualValues(t, 0, fp.calls.Load())
}

func Test_Validation_CachedPerCredential(t *testing.T) {
	fp := newFakePlatform(t)
	ms := newValidatingMicroservice(fp)
	h := ms.TenantContext(TenantContextOptions{UseServiceUser: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	for range 100 {
		require.Equal(t, http.StatusOK, serve(h, basicReq("t123/jane", "good")).Code)
	}
	assert.EqualValues(t, 1, fp.calls.Load())

	for _, user := range []string{"a", "b", "c"} {
		for range 10 {
			serve(h, basicReq("t123/"+user, "good"))
		}
	}
	assert.EqualValues(t, 4, fp.calls.Load(), "one call per distinct credential")
}

func Test_Validation_ConcurrentBurstSharesOneCall(t *testing.T) {
	fp := newFakePlatform(t)
	fp.delay = 50 * time.Millisecond
	ms := newValidatingMicroservice(fp)
	h := ms.TenantContext(TenantContextOptions{UseServiceUser: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			serve(h, basicReq("t123/jane", "good"))
		}()
	}
	wg.Wait()
	assert.EqualValues(t, 1, fp.calls.Load())
}

func Test_Validation_NegativeCache(t *testing.T) {
	fp := newFakePlatform(t)
	ms := newValidatingMicroservice(fp)
	h := ms.TenantContext(TenantContextOptions{UseServiceUser: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	serve(h, basicReq("t123/x", "wrong"))
	serve(h, basicReq("t123/x", "wrong"))
	assert.EqualValues(t, 1, fp.calls.Load(), "failed validations are cached briefly")
}

func Test_Validation_TokenExpiryCapsTTL(t *testing.T) {
	fp := newFakePlatform(t)
	ms := newValidatingMicroservice(fp)
	h := ms.TenantContext(TenantContextOptions{Validation: ValidateCached})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	claims := fmt.Sprintf(`{"ten":"t123","sub":"jane","exp":%d}`, time.Now().Add(time.Second).Unix())
	token := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`)) + "." + base64.RawURLEncoding.EncodeToString([]byte(claims)) + ".sig"
	req := func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/devices", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		return r
	}
	serve(h, req())
	time.Sleep(1100 * time.Millisecond)
	serve(h, req())
	assert.EqualValues(t, 2, fp.calls.Load(), "the token expired, so it is validated again")
}

func Test_Validation_OutageIsNotCached(t *testing.T) {
	fp := newFakePlatform(t)
	fp.status.Store(http.StatusServiceUnavailable)
	ms := newValidatingMicroservice(fp)
	h := ms.TenantContext(TenantContextOptions{UseServiceUser: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	w := serve(h, basicReq("t123/jane", "good"))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	fp.status.Store(0)
	w = serve(h, basicReq("t123/jane", "good"))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.EqualValues(t, 2, fp.calls.Load())
}

func Test_Validation_VerifiedCallerInContext(t *testing.T) {
	fp := newFakePlatform(t)
	ms := newValidatingMicroservice(fp)
	var caller CallerInfo
	var tenant string
	h := ms.TenantContext(TenantContextOptions{Validation: ValidateCached})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller, _ = CallerFromContext(r.Context())
		tenant = TenantFromContext(r.Context())
	}))

	require.Equal(t, http.StatusOK, serve(h, basicReq("t123/admin", "good")).Code)
	assert.Equal(t, "t123", tenant)
	assert.Equal(t, "t123/admin", caller.UserName)
	assert.True(t, caller.HasRoles("ROLE_INVENTORY_ADMIN"))
}

func Test_Validation_CustomValidator(t *testing.T) {
	fp := newFakePlatform(t)
	ms := newValidatingMicroservice(fp)
	var tenant string
	h := ms.TenantContext(TenantContextOptions{
		Validation: ValidateCached,
		Validator: func(ctx context.Context, auth authentication.AuthOptions) (CallerInfo, error) {
			return CallerInfo{Tenant: "t1", UserName: "custom"}, nil
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant = TenantFromContext(r.Context())
	}))

	require.Equal(t, http.StatusOK, serve(h, basicReq("t999/x", "y")).Code)
	assert.Equal(t, "t1", tenant)
	assert.EqualValues(t, 0, fp.calls.Load())
}

func Test_RequireRoles(t *testing.T) {
	fp := newFakePlatform(t)
	ms := newValidatingMicroservice(fp)
	called := 0
	// The surrounding middleware does not validate; RequireRoles must.
	h := ms.TenantContext(TenantContextOptions{Validation: ValidateNone})(
		ms.RequireRoles("ROLE_INVENTORY_ADMIN")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called++
		})))

	w := serve(h, basicReq("t123/jane", "good"))
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, 0, called)

	w = serve(h, basicReq("t123/admin", "good"))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, called)

	w = serve(h, basicReq("t123/admin", "wrong"))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
