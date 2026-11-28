package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/stretchr/testify/assert"
)

func circuitBreakerClient(t *testing.T, cb *CircuitBreakerOptions) (*Client, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	return mustNewClient(ClientOptions{
		BaseURL:          srv.URL,
		SkipTenantLookup: true,
		Retry:            &RetryOptions{Count: 0, CircuitBreaker: cb},
	}), &hits
}

func TestClient_CircuitBreaker_Configurable(t *testing.T) {
	client, hits := circuitBreakerClient(t, &CircuitBreakerOptions{FailureThreshold: 2})
	for i := 0; i < 5; i++ {
		_ = client.ManagedObjects.List(context.Background(), managedobjects.ListOptions{})
	}
	assert.Equal(t, int32(2), hits.Load(), "the breaker opens after 2 failures: later requests are not sent")
}

func TestClient_CircuitBreaker_Disabled(t *testing.T) {
	client, hits := circuitBreakerClient(t, &CircuitBreakerOptions{Disabled: true})
	for i := 0; i < 15; i++ {
		_ = client.ManagedObjects.List(context.Background(), managedobjects.ListOptions{})
	}
	assert.Equal(t, int32(15), hits.Load())
}

func TestClient_CircuitBreaker_DefaultOpensAfterTenFailures(t *testing.T) {
	client, hits := circuitBreakerClient(t, nil)
	for i := 0; i < 15; i++ {
		_ = client.ManagedObjects.List(context.Background(), managedobjects.ListOptions{})
	}
	assert.Equal(t, int32(10), hits.Load())
}
