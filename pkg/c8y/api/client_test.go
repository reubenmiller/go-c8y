package api

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Create a Cumulocity client and use it to query the platform
func Example_newClient() {
	client := NewClientFromEnvironment(ClientOptions{})

	alarmCollection := client.Alarms.List(context.Background(), alarms.ListOptions{
		Severity: []string{
			model.AlarmSeverityMajor,
		},
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
