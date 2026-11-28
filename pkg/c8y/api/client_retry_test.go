package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/stretchr/testify/assert"
	resty "resty.dev/v3"
)

func retryTestClient(t *testing.T, retry *RetryOptions) (*Client, *atomic.Int32) {
	t.Helper()
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	client := mustNewClient(ClientOptions{
		BaseURL:          srv.URL,
		SkipTenantLookup: true,
		Auth:             authentication.AuthOptions{Username: "u", Password: "p", Tenant: "t", AuthType: []authentication.AuthType{authentication.AuthTypeBasic}},
		Retry:            retry,
	})
	return client, &attempts
}

// Retry.Count 0 sends a failing request once: the default of 3 retries (with
// seconds of waiting) is only kept when ClientOptions.Retry is nil.
func TestClientOptions_RetryCountZeroSendsOnce(t *testing.T) {
	client, attempts := retryTestClient(t, &RetryOptions{})
	start := time.Now()
	_ = client.ManagedObjects.List(context.Background(), managedobjects.ListOptions{})
	assert.Equal(t, int32(1), attempts.Load())
	assert.Less(t, time.Since(start), 2*time.Second)
}

func TestClientOptions_RetryCountAndWait(t *testing.T) {
	client, attempts := retryTestClient(t, &RetryOptions{Count: 2, WaitMin: 10 * time.Millisecond, WaitMax: 20 * time.Millisecond})
	start := time.Now()
	_ = client.ManagedObjects.List(context.Background(), managedobjects.ListOptions{})
	assert.Equal(t, int32(3), attempts.Load(), "first attempt and 2 retries")
	assert.Less(t, time.Since(start), 2*time.Second)
}

func TestExponentialRetryDelay(t *testing.T) {
	strategy := exponentialRetryDelay(5*time.Second, 50*time.Second)
	for attempt, want := range map[int]time.Duration{
		1: 5 * time.Second,
		2: 10 * time.Second,
		3: 20 * time.Second,
		4: 40 * time.Second,
		5: 50 * time.Second,
		9: 50 * time.Second,
	} {
		got, err := strategy(newAttemptResponse(attempt), nil)
		assert.NoError(t, err)
		assert.Equal(t, want, got, "retry %d", attempt)
	}
}

func newAttemptResponse(attempt int) *resty.Response {
	return &resty.Response{Request: &resty.Request{Attempt: attempt}}
}

// Exponential waits WaitMin, then twice as long, without jitter, so 2 retries
// against a failing server take WaitMin*3.
func TestClientOptions_RetryExponential(t *testing.T) {
	client, attempts := retryTestClient(t, &RetryOptions{Count: 2, WaitMin: 100 * time.Millisecond, WaitMax: 5 * time.Second, Exponential: true})
	start := time.Now()
	_ = client.ManagedObjects.List(context.Background(), managedobjects.ListOptions{})
	elapsed := time.Since(start)
	assert.Equal(t, int32(3), attempts.Load())
	assert.GreaterOrEqual(t, elapsed, 300*time.Millisecond, "100ms + 200ms")
	assert.Less(t, elapsed, 450*time.Millisecond)
}
