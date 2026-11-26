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
	client := NewClient(ClientOptions{
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
