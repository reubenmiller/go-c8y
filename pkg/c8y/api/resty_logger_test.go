package api

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ClientOptions_Logger(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"1","type":"t","text":"x","source":{"id":"1"}}`))
	}))
	defer srv.Close()

	newClient := func(level slog.Level) (*Client, *bytes.Buffer) {
		var buf bytes.Buffer
		return mustNewClient(ClientOptions{
			BaseURL:          srv.URL,
			SkipTenantLookup: true,
			Auth: authentication.AuthOptions{
				Tenant: "t1", Username: "u", Password: "p",
				AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
			},
			Logger: slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level})),
		}), &buf
	}

	// The plain HTTP warning is an info message of the HTTP layer.
	client, buf := newClient(slog.LevelInfo)
	require.NoError(t, client.Events.Get(context.Background(), "1").Err)
	assert.Contains(t, buf.String(), "Using sensitive credentials in HTTP mode")
	assert.Contains(t, buf.String(), "component=http")

	client, buf = newClient(slog.LevelWarn)
	require.NoError(t, client.Events.Get(context.Background(), "1").Err)
	assert.Empty(t, buf.String(), "a warn-level logger hides the HTTP layer's warnings")
}
