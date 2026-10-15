package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/devices"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"resty.dev/v3"
)

func Test_AuthorizationHeader_MatchesRequestsSentByClient(t *testing.T) {
	basicOpts := authentication.AuthOptions{
		Tenant:   "t100",
		Username: "bootstrap",
		Password: "bootpass",
		AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
	}
	// Each case decorates the subtest's context (nil: the plain context), so no context is stored in the table.
	serviceUser := func(ctx context.Context) context.Context {
		return WithServiceUser(ctx, model.ServiceUser{Tenant: "t200", Username: "service_app", Password: "servicepass"})
	}
	tokenAuth := func(ctx context.Context) context.Context {
		return WithAuth(ctx, authentication.AuthOptions{Tenant: "t300", Token: "ctx-token"})
	}
	skipExchange := func(ctx context.Context) context.Context {
		return ctxhelpers.WithSkipContextTokenExchange(serviceUser(ctx))
	}
	emptyAuth := func(ctx context.Context) context.Context {
		return WithAuth(ctx, authentication.AuthOptions{})
	}

	tests := []struct {
		name     string
		auth     authentication.AuthOptions
		exchange bool
		withCtx  func(context.Context) context.Context
		want     string
	}{
		{"basic", basicOpts, false, nil, basicAuth("t100/bootstrap", "bootpass")},
		{"static token", authentication.AuthOptions{Token: "static-token"}, false, nil, "Bearer static-token"},
		{"token preferred over basic when unset", authentication.AuthOptions{Tenant: "t1", Username: "u", Password: "p", Token: "tok",
			AuthType: []authentication.AuthType{authentication.AuthTypeUnset}}, false, nil, "Bearer tok"},
		{"context service user overrides client", basicOpts, false, serviceUser, basicAuth("t200/service_app", "servicepass")},
		{"context token overrides client", basicOpts, false, tokenAuth, "Bearer ctx-token"},
		{"context exchange", basicOpts, true, serviceUser, "Bearer exchanged-token-1"},
		{"context exchange skipped", basicOpts, true, skipExchange, basicAuth("t200/service_app", "servicepass")},
		{"empty context auth falls back to client", basicOpts, false, emptyAuth, basicAuth("t100/bootstrap", "bootpass")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs, _ := newTokenExchangeServer(http.StatusOK)
			defer rs.server.Close()
			client := NewClient(ClientOptions{BaseURL: rs.server.URL, Auth: tt.auth, ContextTokenExchange: tt.exchange})

			ctx := t.Context()
			if tt.withCtx != nil {
				ctx = tt.withCtx(ctx)
			}
			got, err := client.AuthorizationHeader(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)

			// The header must be the one the client really sends
			require.NoError(t, client.Devices.List(ctx, devices.ListOptions{}).Err)
			assert.Equal(t, got, rs.last(t).Authorization)
		})
	}
}

func Test_AuthorizationHeader_NoCredentials(t *testing.T) {
	client := NewClient(ClientOptions{BaseURL: "http://localhost:1"})
	got, err := client.AuthorizationHeader(context.Background())
	require.NoError(t, err)
	assert.Empty(t, got)

	// An incomplete context auth has no credentials either
	got, err = client.AuthorizationHeader(WithAuth(context.Background(), authentication.AuthOptions{Username: "u"}))
	require.NoError(t, err)
	assert.Empty(t, got)
}

func Test_AuthorizationHeader_NoAuthContext(t *testing.T) {
	client := NewClient(ClientOptions{
		BaseURL: "http://localhost:1",
		Auth:    authentication.AuthOptions{Token: "static-token"},
	})
	got, err := client.AuthorizationHeader(ctxhelpers.WithNoAuth(context.Background()))
	require.NoError(t, err)
	assert.Empty(t, got)
}

func Test_AuthorizationHeader_TokenSourceRenewal(t *testing.T) {
	var issued int64
	source := authentication.NewCachedTokenSource(authentication.TokenSourceFunc(func() (*authentication.Token, error) {
		n := atomic.AddInt64(&issued, 1)
		return &authentication.Token{
			AccessToken: fmt.Sprintf("source-token-%d", n),
			Expiry:      time.Now().Add(time.Hour),
		}, nil
	}))
	client := NewClient(ClientOptions{
		BaseURL: "http://localhost:1",
		Auth:    authentication.AuthOptions{TokenSource: source},
	})
	require.EqualValues(t, 1, atomic.LoadInt64(&issued)) // primed at construction

	got, err := client.AuthorizationHeader(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Bearer source-token-1", got)
	assert.EqualValues(t, 1, atomic.LoadInt64(&issued), "cached token is reused")

	source.Invalidate()
	got, err = client.AuthorizationHeader(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Bearer source-token-2", got, "an invalidated or expired token is renewed")

	// Per-request credentials and skip/dry-run contexts bypass the token source
	got, err = client.AuthorizationHeader(WithAuth(context.Background(), authentication.AuthOptions{Token: "ctx"}))
	require.NoError(t, err)
	assert.Equal(t, "Bearer ctx", got)
	assert.EqualValues(t, 2, atomic.LoadInt64(&issued))
}

func Test_AuthorizationHeader_TokenSourceError(t *testing.T) {
	fail := false
	source := authentication.NewCachedTokenSource(authentication.TokenSourceFunc(func() (*authentication.Token, error) {
		if fail {
			return nil, errors.New("login failed")
		}
		return &authentication.Token{AccessToken: "t", Expiry: time.Now().Add(time.Hour)}, nil
	}))
	client := NewClient(ClientOptions{BaseURL: "http://localhost:1", Auth: authentication.AuthOptions{TokenSource: source}})

	fail = true
	source.Invalidate()
	got, err := client.AuthorizationHeader(context.Background())
	assert.ErrorContains(t, err, "login failed")
	assert.Empty(t, got)
}

// Test_AuthorizationHeader_MatchesWire checks the client-level credentials
// against the header the HTTP client really sends after SetAuth, including a
// token set next to pinned Basic credentials (resty sends the token).
func Test_AuthorizationHeader_MatchesWire(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(HeaderAuthorization)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	basicOnly := []authentication.AuthType{authentication.AuthTypeBasic}
	tests := []struct {
		name string
		auth authentication.AuthOptions
	}{
		{"none", authentication.AuthOptions{}},
		{"basic", authentication.AuthOptions{Tenant: "t1", Username: "u", Password: "p", AuthType: basicOnly}},
		{"basic without password", authentication.AuthOptions{Username: "u", AuthType: basicOnly}},
		{"token", authentication.AuthOptions{Token: "tok", AuthType: []authentication.AuthType{authentication.AuthTypeBearer}}},
		{"token and basic, unset", authentication.AuthOptions{Tenant: "t1", Username: "u", Password: "p", Token: "tok"}},
		{"token and basic, pinned basic", authentication.AuthOptions{Tenant: "t1", Username: "u", Password: "p", Token: "tok", AuthType: basicOnly}},
		{"basic only, unset", authentication.AuthOptions{Username: "u", Password: "p"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(ClientOptions{BaseURL: "http://localhost:1"})
			client.Auth = tt.auth
			rc := resty.New()
			defer rc.Close()
			SetAuth(rc, tt.auth)
			got = ""
			_, err := rc.R().Get(srv.URL)
			require.NoError(t, err)

			header, err := client.AuthorizationHeader(t.Context())
			require.NoError(t, err)
			assert.Equal(t, got, header)
		})
	}
}
