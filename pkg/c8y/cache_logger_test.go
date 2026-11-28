package c8y_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The cache logs through CacheOptions.Logger (the client's logger), not the
// global slog default.
func TestCachedTransport_LogsThroughConfiguredLogger(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	// The global default must stay quiet.
	var global bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&global, nil)))
	defer slog.SetDefault(prev)

	transport := c8y.NewCachedTransport(nil, time.Minute, t.TempDir(), nil, c8y.CacheOptions{Logger: logger})
	client := &http.Client{Transport: transport}

	for range 2 {
		res, err := client.Get(srv.URL + "/inventory/managedObjects")
		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}

	assert.Contains(t, buf.String(), "Using cached response")
	assert.Empty(t, global.String())
}

// NewCachedClient caches the responses of GET requests in the cache directory.
func TestNewCachedClient_CachesGet(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := c8y.NewCachedClient(&http.Client{}, t.TempDir(), time.Minute, nil, c8y.CacheOptions{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	for range 2 {
		res, err := client.Get(srv.URL + "/inventory/managedObjects")
		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}
	assert.Equal(t, 1, hits, "the second request is served from the cache")
}
