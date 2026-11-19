package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mwServer records requests and can fail the first N requests with 503.
type mwServer struct {
	mu        sync.Mutex
	requests  []*http.Request
	bodies    []string
	failFirst int32
	server    *httptest.Server
}

func newMWServer(t *testing.T) *mwServer {
	t.Helper()
	ms := &mwServer{}
	ms.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		ms.mu.Lock()
		ms.requests = append(ms.requests, r.Clone(context.Background()))
		ms.bodies = append(ms.bodies, string(b))
		ms.mu.Unlock()
		if atomic.AddInt32(&ms.failFirst, -1) >= 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"managedObjects":[],"statistics":{"currentPage":1,"pageSize":5},"name":"server"}`))
	}))
	t.Cleanup(ms.server.Close)
	return ms
}

func (ms *mwServer) count() int {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return len(ms.requests)
}

func (ms *mwServer) last(t *testing.T) *http.Request {
	t.Helper()
	ms.mu.Lock()
	defer ms.mu.Unlock()
	require.NotEmpty(t, ms.requests)
	return ms.requests[len(ms.requests)-1]
}

func newMWClient(ms *mwServer, mw ...Middleware) *Client {
	return NewClient(ClientOptions{
		BaseURL:          ms.server.URL,
		SkipTenantLookup: true,
		Auth: authentication.AuthOptions{
			Tenant:   "t100",
			Username: "user",
			Password: "pass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
		Middleware: mw,
	})
}

func get(c *Client, ctx context.Context) *RequestResult {
	return c.SendRequest(ctx, RequestOptions{Method: http.MethodGet, Path: "/inventory/managedObjects"})
}

func Test_Middleware_AddsHeader(t *testing.T) {
	ms := newMWServer(t)
	addHeader := func(next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			r.Header.Set("X-Test", "1")
			return next.RoundTrip(r)
		})
	}
	client := newMWClient(ms, addHeader)

	require.NoError(t, get(client, context.Background()).Error)
	require.NoError(t, get(client, context.Background()).Error)
	assert.Equal(t, 2, ms.count())
	assert.Equal(t, "1", ms.last(t).Header.Get("X-Test"))
}

func Test_Middleware_Order(t *testing.T) {
	ms := newMWServer(t)
	var mu sync.Mutex
	var events []string
	record := func(name string) Middleware {
		return func(next http.RoundTripper) http.RoundTripper {
			return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				mu.Lock()
				events = append(events, name+":request")
				mu.Unlock()
				resp, err := next.RoundTrip(r)
				mu.Lock()
				events = append(events, name+":response")
				mu.Unlock()
				return resp, err
			})
		}
	}
	client := newMWClient(ms, record("A"), record("B"))

	require.NoError(t, get(client, context.Background()).Error)
	assert.Equal(t, []string{"A:request", "B:request", "B:response", "A:response"}, events)
}

func Test_Middleware_UseAfterCreation(t *testing.T) {
	ms := newMWServer(t)
	client := newMWClient(ms)
	var n int64
	client.Use(CountRequests(&n))

	require.NoError(t, get(client, context.Background()).Error)
	assert.EqualValues(t, 1, atomic.LoadInt64(&n))
}

func Test_CountRequests(t *testing.T) {
	ms := newMWServer(t)
	var n int64
	client := newMWClient(ms, CountRequests(&n))

	for range 3 {
		require.NoError(t, get(client, context.Background()).Error)
	}
	assert.EqualValues(t, 3, atomic.LoadInt64(&n))
}

func Test_CountRequests_CountsRetryAttempts(t *testing.T) {
	ms := newMWServer(t)
	ms.failFirst = 1
	var n int64
	client := newMWClient(ms, CountRequests(&n))
	client.Resty().SetRetryWaitTime(0).SetRetryMaxWaitTime(0)

	result := get(client, context.Background())
	require.NoError(t, result.Error)
	assert.Equal(t, http.StatusOK, result.StatusCode())
	assert.Equal(t, 2, ms.count(), "the server should see the failed attempt and the retry")
	assert.EqualValues(t, 2, atomic.LoadInt64(&n))
}

func Test_Counters_IgnoreDryRun(t *testing.T) {
	ms := newMWServer(t)
	var n int64
	stats := NewStatsMap()
	client := newMWClient(ms, CountRequests(&n), CountByMethodAndPath(stats))

	_ = get(client, WithDryRun(context.Background(), true))
	assert.EqualValues(t, 0, atomic.LoadInt64(&n))
	assert.Equal(t, 0, ms.count())
	assert.EqualValues(t, 0, stats.Get(http.MethodGet, "/inventory/managedObjects"))
}

func Test_CountByMethodAndPath(t *testing.T) {
	ms := newMWServer(t)
	stats := NewStatsMap()
	client := newMWClient(ms, CountByMethodAndPath(stats))

	require.NoError(t, get(client, context.Background()).Error)
	require.NoError(t, get(client, context.Background()).Error)
	assert.EqualValues(t, 2, stats.Get(http.MethodGet, "/inventory/managedObjects"))
}

func Test_PrepareRequest_ChangesRequest(t *testing.T) {
	ms := newMWServer(t)
	client := newMWClient(ms)

	result := client.SendRequest(context.Background(), RequestOptions{
		Method: http.MethodGet,
		Path:   "/inventory/managedObjects",
		PrepareRequest: func(r *http.Request) error {
			q := r.URL.Query()
			q.Set("withParents", "true")
			r.URL.RawQuery = q.Encode()
			return nil
		},
	})
	require.NoError(t, result.Error)
	assert.Equal(t, "true", ms.last(t).URL.Query().Get("withParents"))
}

func Test_PrepareRequest_ErrorAbortsRequest(t *testing.T) {
	ms := newMWServer(t)
	client := newMWClient(ms)
	client.Resty().SetRetryCount(0)
	hookErr := errors.New("hook failed")

	result := client.SendRequest(context.Background(), RequestOptions{
		Method:         http.MethodGet,
		Path:           "/inventory/managedObjects",
		PrepareRequest: func(r *http.Request) error { return hookErr },
	})
	require.Error(t, result.Error)
	assert.ErrorIs(t, result.Error, hookErr)
	assert.Equal(t, 0, ms.count(), "no request should reach the server")
}

func Test_OnResponse_ReadingBodyKeepsCallerBody(t *testing.T) {
	ms := newMWServer(t)
	client := newMWClient(ms)

	var hookBody string
	result := client.SendRequest(context.Background(), RequestOptions{
		Method: http.MethodGet,
		Path:   "/inventory/managedObjects",
		OnResponse: func(resp *http.Response) error {
			b, err := io.ReadAll(resp.Body)
			hookBody = string(b)
			return err
		},
	})
	require.NoError(t, result.Error)
	assert.Contains(t, hookBody, `"name":"server"`)
	assert.Contains(t, result.String(), `"name":"server"`)
	callerBody, err := io.ReadAll(result.Response.Body)
	require.NoError(t, err)
	assert.Contains(t, string(callerBody), `"name":"server"`)
}

func Test_RequestResult_RequestBodyStaysStable(t *testing.T) {
	ms := newMWServer(t)
	client := newMWClient(ms)

	kept := client.SendRequest(context.Background(), RequestOptions{
		Method: http.MethodPost,
		Path:   "/inventory/managedObjects",
		Body:   map[string]any{"name": "original"},
	})
	require.NoError(t, kept.Error)
	require.NotNil(t, kept.Response.Request)

	for i := range 100 {
		r := client.SendRequest(context.Background(), RequestOptions{
			Method: http.MethodPost,
			Path:   "/inventory/managedObjects",
			Body:   map[string]any{"name": fmt.Sprintf("other-%d", i)},
		})
		require.NoError(t, r.Error)
	}

	body, err := io.ReadAll(kept.Response.Request.Body)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(body), `"original"`), "kept request body changed: %s", body)
}

func Test_Resty_EscapeHatch(t *testing.T) {
	ms := newMWServer(t)
	client := newMWClient(ms)
	require.NotNil(t, client.Resty())
	assert.Equal(t, strings.TrimSuffix(ms.server.URL, "/"), strings.TrimSuffix(client.Resty().BaseURL(), "/"))
}
