package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	oauth2_api "github.com/reubenmiller/go-c8y/v2/pkg/oauth/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/oauth/device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// authServer records the Authorization header of every request. The token
// endpoint hands out "jwt-1", "jwt-2", ... or fails when tokenStatus is set.
type authServer struct {
	*httptest.Server
	mu          sync.Mutex
	reqs        []authRecord
	tokenStatus int
	tokens      int
}

type authRecord struct {
	Method, Path, Authorization string
}

func newAuthServer(t *testing.T) *authServer {
	t.Helper()
	s := &authServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.reqs = append(s.reqs, authRecord{r.Method, r.URL.Path, r.Header.Get("Authorization")})
		status := s.tokenStatus
		if r.URL.Path == "/tenant/oauth/token" {
			s.tokens++
		}
		n := s.tokens
		s.mu.Unlock()
		if r.URL.Path == "/tenant/oauth/token" {
			if status != 0 {
				w.WriteHeader(status)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"` + jwtFor(n) + `","token_type":"bearer"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(s.Close)
	return s
}

func jwtFor(n int) string {
	return "jwt-" + string(rune('0'+n))
}

func (s *authServer) calls() []authRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]authRecord(nil), s.reqs...)
}

func (s *authServer) last() string {
	c := s.calls()
	if len(c) == 0 {
		return ""
	}
	return c[len(c)-1].Authorization
}

func (s *authServer) api(path string) []authRecord {
	var out []authRecord
	for _, c := range s.calls() {
		if c.Path == path {
			out = append(out, c)
		}
	}
	return out
}

func (s *authServer) newClient(auth authentication.AuthOptions, eager bool) *Client {
	return mustNewClient(ClientOptions{BaseURL: s.URL, SkipTenantLookup: true, Auth: auth, EagerLogin: eager})
}

const probePath = "/inventory/managedObjects"

func probe(c *Client, ctx context.Context) op.Result[jsondoc.JSONDoc] {
	return c.Request(ctx, RequestOptions{Method: http.MethodGet, Path: probePath})
}

func basicHeader(user, pass string) string {
	return basicAuthorization(user, pass)
}

func Test_SetAuth_ClearsPreviousCredentials(t *testing.T) {
	tests := []struct {
		name   string
		first  authentication.AuthOptions
		second authentication.AuthOptions
		want   string
	}{
		{
			name:   "bearer token to basic",
			first:  authentication.AuthOptions{Token: "old-token"},
			second: authentication.AuthOptions{Username: "u", Password: "p", AuthType: []authentication.AuthType{authentication.AuthTypeBasic}},
			want:   basicHeader("u", "p"),
		},
		{
			name:   "bearer token to other bearer token",
			first:  authentication.AuthOptions{Token: "old-token"},
			second: authentication.AuthOptions{Token: "new-token"},
			want:   "Bearer new-token",
		},
		{
			name:   "bearer token to nothing",
			first:  authentication.AuthOptions{Token: "old-token"},
			second: authentication.AuthOptions{},
			want:   "",
		},
		{
			name:   "basic to nothing",
			first:  authentication.AuthOptions{Username: "u", Password: "p", AuthType: []authentication.AuthType{authentication.AuthTypeBasic}},
			second: authentication.AuthOptions{},
			want:   "",
		},
		{
			name:   "basic to bearer token",
			first:  authentication.AuthOptions{Username: "u", Password: "p", AuthType: []authentication.AuthType{authentication.AuthTypeBasic}},
			second: authentication.AuthOptions{Token: "tok"},
			want:   "Bearer tok",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newAuthServer(t)
			client := srv.newClient(tt.first, false)
			require.NoError(t, probe(client, t.Context()).Err)
			assert.NotEmpty(t, srv.last(), "the first credentials are used")

			client.SetAuth(tt.second)
			require.NoError(t, probe(client, t.Context()).Err)
			assert.Equal(t, tt.want, srv.last())
		})
	}
}

func Test_SetAuth_DropsTokenOfPreviousTokenSource(t *testing.T) {
	srv := newAuthServer(t)
	// username and password: the client exchanges them for jwt-1
	client := srv.newClient(authentication.AuthOptions{Username: "u", Password: "p"}, false)
	require.NoError(t, probe(client, t.Context()).Err)
	assert.Equal(t, "Bearer jwt-1", srv.last())

	// a different static token must replace the exchanged one
	client.SetAuth(authentication.AuthOptions{Token: "static"})
	require.NoError(t, probe(client, t.Context()).Err)
	assert.Equal(t, "Bearer static", srv.last())

	// and back to credentials: a new exchange, not a cached token of the old source
	client.SetAuth(authentication.AuthOptions{Username: "u2", Password: "p2"})
	require.NoError(t, probe(client, t.Context()).Err)
	assert.Equal(t, "Bearer jwt-2", srv.last())
}

func Test_NewClient_DoesNoNetworkIO(t *testing.T) {
	srv := newAuthServer(t)
	creds := authentication.AuthOptions{Tenant: "t1", Username: "u", Password: "p"}

	client := srv.newClient(creds, false)
	assert.Empty(t, srv.calls(), "NewClient with credentials must not send requests")

	// the token is requested on the first request, once
	require.NoError(t, probe(client, t.Context()).Err)
	require.NoError(t, probe(client, t.Context()).Err)
	assert.Len(t, srv.api("/tenant/oauth/token"), 1)
	assert.Equal(t, "Bearer jwt-1", srv.last())
}

func Test_NewClient_LazyWithBasicAuthType(t *testing.T) {
	srv := newAuthServer(t)
	client := srv.newClient(authentication.AuthOptions{
		Username: "u", Password: "p",
		AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
	}, false)
	require.NoError(t, probe(client, t.Context()).Err)
	assert.Empty(t, srv.api("/tenant/oauth/token"), "AuthType basic never exchanges a token")
	assert.Equal(t, basicHeader("u", "p"), srv.last())
}

func Test_NewClient_EagerLoginOptIn(t *testing.T) {
	srv := newAuthServer(t)
	client := srv.newClient(authentication.AuthOptions{Username: "u", Password: "p"}, true)
	assert.Len(t, srv.api("/tenant/oauth/token"), 1, "EagerLogin requests the token in NewClient")
	assert.Equal(t, "jwt-1", client.Auth().Token)

	require.NoError(t, probe(client, t.Context()).Err)
	assert.Len(t, srv.api("/tenant/oauth/token"), 1)
	assert.Equal(t, "Bearer jwt-1", srv.last())
}

func Test_NewClient_EagerLoginFailureIsNotFatal(t *testing.T) {
	srv := newAuthServer(t)
	srv.tokenStatus = http.StatusUnauthorized
	client := srv.newClient(authentication.AuthOptions{Username: "u", Password: "p"}, true)
	require.NoError(t, probe(client, t.Context()).Err)
	// no token: basic credentials take over
	assert.Equal(t, basicHeader("u", "p"), srv.last())
}

func Test_Login_ReusesTokenForRequests(t *testing.T) {
	srv := newAuthServer(t)
	client := srv.newClient(authentication.AuthOptions{Username: "u", Password: "p"}, false)
	tok, err := client.Login(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "jwt-1", tok)
	require.NoError(t, probe(client, t.Context()).Err)
	assert.Len(t, srv.api("/tenant/oauth/token"), 1, "the request uses the token of Login")
	assert.Equal(t, "Bearer jwt-1", srv.last())
}

// deviceFlowServer is an identity provider for the device flow plus the
// Cumulocity probe endpoint.
func deviceFlowServer(t *testing.T) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/init":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"redirectTo":"` + srv.URL + `/authorize?client_id=cid"}`))
		case "/device":
			w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
			_, _ = w.Write([]byte(url.Values{
				"device_code": {"dc"}, "user_code": {"uc"}, "verification_uri": {srv.URL + "/verify"},
				"interval": {"0"}, "expires_in": {"30"},
			}.Encode()))
		case "/token":
			w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
			_, _ = w.Write([]byte(url.Values{"access_token": {"flow-token"}, "token_type": {"bearer"}}.Encode()))
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func Test_AuthorizeWithDeviceFlow_OnlySetsToken(t *testing.T) {
	idp := deviceFlowServer(t)
	var seen []string
	var mu sync.Mutex
	wrapped := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == probePath {
			mu.Lock()
			seen = append(seen, r.Header.Get("Authorization"))
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer wrapped.Close()

	original := authentication.AuthOptions{
		Tenant: "t1", Username: "u", Password: "p",
		AuthType: []authentication.AuthType{authentication.AuthTypeBearer},
	}
	client := mustNewClient(ClientOptions{BaseURL: wrapped.URL, SkipTenantLookup: true, Auth: original})

	tok, err := client.AuthorizeWithDeviceFlow(t.Context(), idp.URL+"/init", oauth2_api.AuthEndpoints{
		TokenURL:               idp.URL + "/token",
		DeviceAuthorizationURL: idp.URL + "/device",
	}, func(*device.CodeResponse) error { return nil })
	require.NoError(t, err)
	assert.Equal(t, "flow-token", tok.Token)

	assert.Equal(t, "flow-token", client.Auth().Token)
	assert.Equal(t, "t1", client.Auth().Tenant)
	assert.Equal(t, "u", client.Auth().Username)
	assert.Equal(t, "p", client.Auth().Password)
	assert.Equal(t, original.AuthType, client.Auth().AuthType)

	require.NoError(t, probe(client, t.Context()).Err)
	assert.Equal(t, []string{"Bearer flow-token"}, seen)

	// SetAuth with the previous values restores the old credentials
	client.SetAuth(original)
	assert.Empty(t, client.Auth().Token)
}

func Test_AuthType_HonouredPerRequest(t *testing.T) {
	both := authentication.AuthOptions{Username: "u", Password: "p", Token: "tok"}
	with := func(t authentication.AuthType) authentication.AuthOptions {
		a := both
		a.AuthType = []authentication.AuthType{t}
		return a
	}
	tests := []struct {
		name string
		auth authentication.AuthOptions
		want string
	}{
		{"unset sends the token", both, "Bearer tok"},
		{"bearer sends the token", with(authentication.AuthTypeBearer), "Bearer tok"},
		{"basic sends basic despite the token", with(authentication.AuthTypeBasic), basicHeader("u", "p")},
		{"none sends nothing", with(authentication.AuthTypeNone), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newAuthServer(t)
			// the token endpoint is unavailable: only static credentials apply
			srv.tokenStatus = http.StatusNotFound
			client := srv.newClient(authentication.AuthOptions{}, false)
			client.SetAuth(tt.auth)
			require.NoError(t, probe(client, t.Context()).Err)
			assert.Equal(t, tt.want, srv.last())

			// the Authorization header helper (websocket) agrees
			got, err := client.AuthorizationHeader(t.Context())
			require.NoError(t, err)
			if tt.name != "unset sends the token" {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_AuthType_AppliesAfterSetAuth(t *testing.T) {
	srv := newAuthServer(t)
	creds := authentication.AuthOptions{Username: "u", Password: "p", Token: "tok"}
	client := srv.newClient(creds, false)

	basic := creds
	basic.AuthType = []authentication.AuthType{authentication.AuthTypeBasic}
	client.SetAuth(basic)
	require.NoError(t, probe(client, t.Context()).Err)
	assert.Equal(t, basicHeader("u", "p"), srv.last())

	// switch the preference back without changing credentials
	bearer := creds
	bearer.AuthType = []authentication.AuthType{authentication.AuthTypeBearer}
	srv.tokenStatus = http.StatusNotFound
	client.SetAuth(bearer)
	require.NoError(t, probe(client, t.Context()).Err)
	assert.Equal(t, "Bearer tok", srv.last())
}

func Test_AuthType_RequestOverridesClient(t *testing.T) {
	srv := newAuthServer(t)
	client := srv.newClient(authentication.AuthOptions{
		Username: "u", Password: "p",
		AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
	}, false)

	run := func(typ authentication.AuthType) string {
		rc := client.Resty()
		req := httpcore.NewTryRequest(rc, rc.R()).WithAuthType(typ)
		req.Request.Method = http.MethodGet
		req.Request.URL = probePath
		_, err := httpcore.ExecuteResponseOnly(t.Context(), req)
		require.NoError(t, err)
		return srv.last()
	}
	assert.Equal(t, "Bearer jwt-1", run(authentication.AuthTypeBearer), "a request can ask for a bearer token")
	assert.Equal(t, "", run(authentication.AuthTypeNone))
	assert.Equal(t, basicHeader("u", "p"), run(authentication.AuthTypeBasic))

	// without a type of its own the request follows the client
	require.NoError(t, probe(client, t.Context()).Err)
	assert.Equal(t, basicHeader("u", "p"), srv.last())
}

func Test_AuthType_PerRequestContextAuth(t *testing.T) {
	srv := newAuthServer(t)
	srv.tokenStatus = http.StatusNotFound
	client := srv.newClient(authentication.AuthOptions{Token: "client-token"}, false)

	ctx := WithAuth(t.Context(), authentication.AuthOptions{
		Username: "u", Password: "p", Token: "ctx-token",
		AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
	})
	require.NoError(t, probe(client, ctx).Err)
	assert.Equal(t, basicHeader("u", "p"), srv.last())

	ctx = WithAuth(t.Context(), authentication.AuthOptions{
		Token: "ctx-token", AuthType: []authentication.AuthType{authentication.AuthTypeNone},
	})
	require.NoError(t, probe(client, ctx).Err)
	assert.Equal(t, "", srv.last())

	ctx = WithAuth(t.Context(), authentication.AuthOptions{Token: "ctx-token"})
	require.NoError(t, probe(client, ctx).Err)
	assert.Equal(t, "Bearer ctx-token", srv.last())
}

func Test_RevokedToken_ConcurrentRequestsRefreshOnce(t *testing.T) {
	var mu sync.Mutex
	issued := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/tenant/oauth/token" {
			mu.Lock()
			issued++
			n := issued
			mu.Unlock()
			_, _ = w.Write([]byte(`{"access_token":"` + jwtFor(n) + `"}`))
			return
		}
		if r.Header.Get("Authorization") == "Bearer jwt-1" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"general/internalError","message":"Token is terminated"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	client := mustNewClient(ClientOptions{BaseURL: srv.URL, SkipTenantLookup: true,
		Auth: authentication.AuthOptions{Username: "u", Password: "p"}})
	_, err := client.currentTokenSource().Token() // jwt-1 is cached and about to be revoked
	require.NoError(t, err)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = probe(client, context.Background())
		}()
	}
	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 2, issued, "one refresh for all concurrent 401s of the same token")
}
