package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	serviceUser := WithServiceUser(context.Background(), model.ServiceUser{
		Tenant: "t200", Username: "service_app", Password: "servicepass",
	})
	tokenCtx := WithAuth(context.Background(), authentication.AuthOptions{Tenant: "t300", Token: "ctx-token"})

	tests := []struct {
		name     string
		auth     authentication.AuthOptions
		exchange bool
		ctx      context.Context
		want     string
	}{
		{"basic", basicOpts, false, context.Background(), basicAuth("t100/bootstrap", "bootpass")},
		{"static token", authentication.AuthOptions{Token: "static-token"}, false, context.Background(), "Bearer static-token"},
		{"token preferred over basic when unset", authentication.AuthOptions{Tenant: "t1", Username: "u", Password: "p", Token: "tok",
			AuthType: []authentication.AuthType{authentication.AuthTypeUnset}}, false, context.Background(), "Bearer tok"},
		{"context service user overrides client", basicOpts, false, serviceUser, basicAuth("t200/service_app", "servicepass")},
		{"context token overrides client", basicOpts, false, tokenCtx, "Bearer ctx-token"},
		{"context exchange", basicOpts, true, serviceUser, "Bearer exchanged-token-1"},
		{"context exchange skipped", basicOpts, true, ctxhelpers.WithSkipContextTokenExchange(serviceUser), basicAuth("t200/service_app", "servicepass")},
		{"empty context auth falls back to client", basicOpts, false, WithAuth(context.Background(), authentication.AuthOptions{}), basicAuth("t100/bootstrap", "bootpass")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs, _ := newTokenExchangeServer(http.StatusOK)
			defer rs.server.Close()
			client := NewClient(ClientOptions{BaseURL: rs.server.URL, Auth: tt.auth, ContextTokenExchange: tt.exchange})

			got, err := client.AuthorizationHeader(tt.ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)

			// The header must be the one the client really sends
			require.NoError(t, client.Devices.List(tt.ctx, devices.ListOptions{}).Err)
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

// Test_AuthorizationHeader_AgreesWithMiddlewareAuthorization guards against the
// two drifting apart for the client-level credentials.
func Test_AuthorizationHeader_AgreesWithMiddlewareAuthorization(t *testing.T) {
	tests := []struct {
		name string
		auth authentication.AuthOptions
	}{
		{"none", authentication.AuthOptions{}},
		{"basic", authentication.AuthOptions{Tenant: "t1", Username: "u", Password: "p", AuthType: []authentication.AuthType{authentication.AuthTypeBasic}}},
		{"basic without password", authentication.AuthOptions{Username: "u", AuthType: []authentication.AuthType{authentication.AuthTypeBasic}}},
		{"bearer", authentication.AuthOptions{Token: "tok", AuthType: []authentication.AuthType{authentication.AuthTypeBearer}}},
		{"bearer without token", authentication.AuthOptions{Username: "u", Password: "p", AuthType: []authentication.AuthType{authentication.AuthTypeBearer}}},
		{"unset prefers token", authentication.AuthOptions{Tenant: "t1", Username: "u", Password: "p", Token: "tok"}},
		{"unset basic only", authentication.AuthOptions{Username: "u", Password: "p"}},
		{"auth type none", authentication.AuthOptions{Token: "tok", AuthType: []authentication.AuthType{authentication.AuthTypeNone}}},
		{"bearer then basic", authentication.AuthOptions{Username: "u", Password: "p",
			AuthType: []authentication.AuthType{authentication.AuthTypeBearer, authentication.AuthTypeBasic}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(ClientOptions{BaseURL: "http://localhost:1"})
			client.Auth = tt.auth

			req := resty.New().R()
			require.NoError(t, MiddlewareAuthorization(tt.auth)(nil, req))

			got, err := client.AuthorizationHeader(context.Background())
			require.NoError(t, err)
			assert.Equal(t, req.Header.Get(HeaderAuthorization), got)
		})
	}
}
