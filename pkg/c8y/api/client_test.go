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
	client := NewClientFromEnvironment(ClientOptions{})

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
		client := NewClientFromEnvironment(ClientOptions{})
		assert.Equal(t, testcase.scheme, client.BaseURL.Scheme)
		assert.Equal(t, testcase.host, client.BaseURL.Host)
		assert.Equal(t, testcase.path, client.BaseURL.Path)

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
	client := NewClient(tenantTestOptions(srv.URL, false))
	assert.Equal(t, int32(1), calls.Load())
	assert.Equal(t, "t123", client.Auth.Tenant)
}

func Test_NewClient_SkipTenantLookup(t *testing.T) {
	srv, calls := newTenantServer(t, http.StatusOK)
	client := NewClient(tenantTestOptions(srv.URL, true))
	assert.Equal(t, int32(0), calls.Load())
	assert.Empty(t, client.Auth.Tenant)

	require.NoError(t, client.ResolveTenant(context.Background()))
	assert.Equal(t, int32(1), calls.Load())
	assert.Equal(t, "t123", client.Auth.Tenant)
	assert.Equal(t, "t123", client.HTTPClient.PathParams()[core.PathParamTenantID])
}

func Test_ResolveTenant_Error(t *testing.T) {
	srv, _ := newTenantServer(t, http.StatusUnauthorized)
	client := NewClient(tenantTestOptions(srv.URL, true))
	assert.Error(t, client.ResolveTenant(context.Background()))
	assert.Empty(t, client.Auth.Tenant)
}
