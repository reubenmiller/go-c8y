package api

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Create a Cumulocity client and use it to query the platform
func Example_newClient() {
	client, err := NewClientFromEnvironment(ClientOptions{})
	if err != nil {
		panic(err)
	}

	alarmCollection := client.Alarms.List(context.Background(), alarms.ListOptions{
		Severity: model.NewAlarmSeverity(model.AlarmSeverityMajor),
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 100,
		},
	})
	if alarmCollection.Err != nil {
		panic(alarmCollection.Err)
	}
	for alarm := range jsondoc.DecodeIter[model.Alarm](alarmCollection.Data.Iter()) {
		slog.Info("alarm", "id", alarm.ID, "text", alarm.Text)
	}
}

func Test_formatBase(t *testing.T) {
	assert.Equal(t, "https://example.com/", FormatBaseURL("example.com"))
	assert.Equal(t, "https://example.com/", FormatBaseURL("https://example.com"))
	assert.Equal(t, "http://example.com/", FormatBaseURL("http://example.com"))
	assert.Equal(t, "http://example.com/", FormatBaseURL("http://example.com/"))
	assert.Equal(t, "http://example.com/foo/", FormatBaseURL("http://example.com/foo"))
}

func Test_NewClientFromEnvironment(t *testing.T) {

	cases := []struct {
		env    string
		scheme string
		host   string
		path   string
	}{
		{"example.com", "https", "example.com", "/"},
		{"https://example.com", "https", "example.com", "/"},
		{"http://example.com", "http", "example.com", "/"},
		{"http://example.com/", "http", "example.com", "/"},

		{"http://example.com/foo", "http", "example.com", "/foo/"},
		{"http://example.com/foo/", "http", "example.com", "/foo/"},
	}

	for _, testcase := range cases {
		t.Setenv("C8Y_HOST", testcase.env)
		client, err := NewClientFromEnvironment(ClientOptions{})
		require.NoError(t, err)
		assert.Equal(t, testcase.scheme, client.BaseURL().Scheme)
		assert.Equal(t, testcase.host, client.BaseURL().Host)
		assert.Equal(t, testcase.path, client.BaseURL().Path)

		req := client.common.Client.R().SetMethod(http.MethodGet).SetURL("inventory/managedObjects").SetContext(WithDryRun(context.Background(), true))
		resp, err := req.Send()
		require.NoError(t, err)
		require.NotNil(t, resp.Request.RawRequest)

		expectedPath := testcase.path + "inventory/managedObjects"
		assert.Equal(t, expectedPath, resp.Request.RawRequest.URL.Path)
	}
}

// newTenantServer returns a test server that counts /tenant/currentTenant calls.
func newTenantServer(t *testing.T, status int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tenant/currentTenant" {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"name":"t123"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func tenantTestOptions(baseURL string, skip bool) ClientOptions {
	return ClientOptions{
		BaseURL:          baseURL,
		SkipTenantLookup: skip,
		Auth: authentication.AuthOptions{
			Token: "static-token",
		},
	}
}

func Test_NewClient_TenantLookup(t *testing.T) {
	srv, calls := newTenantServer(t, http.StatusOK)
	client := mustNewClient(tenantTestOptions(srv.URL, false))
	assert.Equal(t, int32(1), calls.Load())
	assert.Equal(t, "t123", client.Auth().Tenant)
}

func Test_NewClient_SkipTenantLookup(t *testing.T) {
	srv, calls := newTenantServer(t, http.StatusOK)
	client := mustNewClient(tenantTestOptions(srv.URL, true))
	assert.Equal(t, int32(0), calls.Load())
	assert.Empty(t, client.Auth().Tenant)

	require.NoError(t, client.ResolveTenant(context.Background()))
	assert.Equal(t, int32(1), calls.Load())
	assert.Equal(t, "t123", client.Auth().Tenant)
	assert.Equal(t, "t123", client.httpClient.PathParams()[core.PathParamTenantID])
}

func Test_ResolveTenant_Error(t *testing.T) {
	srv, _ := newTenantServer(t, http.StatusUnauthorized)
	client := mustNewClient(tenantTestOptions(srv.URL, true))
	assert.Error(t, client.ResolveTenant(context.Background()))
	assert.Empty(t, client.Auth().Tenant)
}

func Test_NewClient_InvalidBaseURL_ReturnsError(t *testing.T) {
	for _, baseURL := range []string{"", "  ", "http://[::1", "http://%zz bad", "https://"} {
		var client *Client
		var err error
		assert.NotPanics(t, func() {
			client, err = NewClient(ClientOptions{BaseURL: baseURL, SkipTenantLookup: true})
		}, baseURL)
		assert.Nil(t, client, baseURL)
		assert.ErrorIs(t, err, ErrInvalidBaseURL, baseURL)
	}
}

func Test_NewClientFromEnvironment_MissingHost_ReturnsError(t *testing.T) {
	for _, name := range authentication.EnvironmentHost {
		t.Setenv(name, "")
	}
	for _, name := range authentication.EnvironmentToken {
		t.Setenv(name, "")
	}
	client, err := NewClientFromEnvironment(ClientOptions{SkipTenantLookup: true})
	assert.Nil(t, client)
	assert.ErrorIs(t, err, ErrInvalidBaseURL)
}

// The environment only fills what the caller left empty.
func Test_NewClientFromEnvironment_KeepsCallerOptions(t *testing.T) {
	t.Setenv("C8Y_HOST", "env.example.com")
	t.Setenv("C8Y_TENANT", "env-tenant")
	t.Setenv("C8Y_USER", "env-user")
	t.Setenv("C8Y_PASSWORD", "env-password")
	t.Setenv("C8Y_TOKEN", "")

	t.Run("caller host and credentials win", func(t *testing.T) {
		client, err := NewClientFromEnvironment(ClientOptions{
			BaseURL:          "caller.example.com",
			Auth:             authentication.AuthOptions{Username: "caller-user", Password: "caller-password"},
			SkipTenantLookup: true,
		})
		require.NoError(t, err)
		assert.Equal(t, "caller.example.com", client.BaseURL().Host)
		auth := client.Auth()
		assert.Equal(t, "caller-user", auth.Username)
		assert.Equal(t, "caller-password", auth.Password)
		assert.Equal(t, "env-tenant", auth.Tenant, "the tenant is the only value taken from the environment")
	})

	t.Run("the environment fills an empty host and empty credentials", func(t *testing.T) {
		client, err := NewClientFromEnvironment(ClientOptions{SkipTenantLookup: true})
		require.NoError(t, err)
		assert.Equal(t, "env.example.com", client.BaseURL().Host)
		auth := client.Auth()
		assert.Equal(t, "env-user", auth.Username)
		assert.Equal(t, "env-password", auth.Password)
		assert.Equal(t, "env-tenant", auth.Tenant)
	})
}

// BaseURL and Auth are read while SetBaseURL, SetAuth and SetToken change
// them; run with -race.
func Test_Client_StateAccessors_AreRaceFree(t *testing.T) {
	client := mustNewClient(ClientOptions{BaseURL: "http://127.0.0.1:1", SkipTenantLookup: true})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			_ = client.SetBaseURL("http://127.0.0.1:2")
			client.SetAuth(authentication.AuthOptions{Tenant: "t", Username: "u", Password: "p"})
			client.SetToken("token")
		}
	}()
	for i := 0; i < 200; i++ {
		_ = client.BaseURL().Host
		_ = client.Auth().Token
	}
	<-done
}

func Test_Client_BaseURL_ReturnsCopy(t *testing.T) {
	client := mustNewClient(ClientOptions{BaseURL: "http://127.0.0.1:1", SkipTenantLookup: true})
	client.BaseURL().Host = "changed.example.com"
	assert.Equal(t, "127.0.0.1:1", client.BaseURL().Host)
}
