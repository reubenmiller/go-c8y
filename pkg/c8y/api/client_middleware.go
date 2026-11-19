package api

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync/atomic"

	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"resty.dev/v3"
)

// Middleware wraps the HTTP transport used by the client. It is the standard
// Go way to observe or change requests and responses:
//
//	addHeader := func(next http.RoundTripper) http.RoundTripper {
//		return api.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
//			r.Header.Set("X-Example", "1")
//			return next.RoundTrip(r)
//		})
//	}
//	client := api.NewClient(api.ClientOptions{Middleware: []api.Middleware{addHeader}})
//
// Middleware runs once per attempt: when a request is retried, it sees each
// attempt. Dry-run and mock requests are answered before the middleware chain,
// so middleware never sees them.
//
// Middleware registered first sees the request first and the response last.
type Middleware func(next http.RoundTripper) http.RoundTripper

// RoundTripperFunc adapts a function to an http.RoundTripper.
type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls f(r).
func (f RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// middlewareTransport applies the middleware chain while keeping the base
// transport reachable, so TLS settings can still be read and changed.
type middlewareTransport struct {
	chain http.RoundTripper
	base  http.RoundTripper
}

func (m *middlewareTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return m.chain.RoundTrip(r)
}

// BaseTransport returns the transport below the middleware chain.
func (m *middlewareTransport) BaseTransport() http.RoundTripper {
	return m.base
}

// TLSClientConfig returns the TLS configuration of the base transport, if any.
func (m *middlewareTransport) TLSClientConfig() *tls.Config {
	switch t := m.base.(type) {
	case *http.Transport:
		return t.TLSClientConfig
	case interface{ TLSClientConfig() *tls.Config }:
		return t.TLSClientConfig()
	}
	return nil
}

// SetTLSClientConfig sets the TLS configuration on the base transport.
func (m *middlewareTransport) SetTLSClientConfig(config *tls.Config) error {
	switch t := m.base.(type) {
	case *http.Transport:
		t.TLSClientConfig = config
		return nil
	case interface{ SetTLSClientConfig(*tls.Config) error }:
		return t.SetTLSClientConfig(config)
	}
	return nil
}

// chainMiddleware wraps base with the given middleware. The first middleware is
// the outermost one.
func chainMiddleware(base http.RoundTripper, middleware []Middleware) http.RoundTripper {
	if len(middleware) == 0 {
		return base
	}
	rt := base
	for i := len(middleware) - 1; i >= 0; i-- {
		if middleware[i] != nil {
			rt = middleware[i](rt)
		}
	}
	return &middlewareTransport{chain: rt, base: base}
}

// Use adds middleware to the client. Call it before sending requests: changing
// the middleware while requests are in flight is not safe.
func (c *Client) Use(middleware ...Middleware) {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()
	c.middleware = append(c.middleware, middleware...)
	if c.dryRunTransport != nil {
		c.dryRunTransport.Transport = chainMiddleware(c.baseTransport, c.middleware)
	}
}

// Resty returns the underlying resty client.
//
// The returned client is NOT covered by the go-c8y v2 compatibility guarantee:
// resty's API can change between go-c8y releases. Prefer Middleware,
// RequestOptions hooks, or context options (WithExtraHeaders, …) where they are
// enough.
func (c *Client) Resty() *resty.Client {
	return c.httpClient
}

func isNotSent(ctx context.Context) bool {
	return ctxhelpers.IsDryRun(ctx) || ctxhelpers.IsMockResponses(ctx)
}

// CountRequests returns middleware that adds 1 to counter for every request
// attempt sent to the server. Retries count as separate attempts; dry-run and
// mock requests are not counted.
func CountRequests(counter *int64) Middleware {
	return func(next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if !isNotSent(r.Context()) {
				atomic.AddInt64(counter, 1)
			}
			return next.RoundTrip(r)
		})
	}
}

// CountByMethodAndPath returns middleware that counts request attempts by HTTP
// method and URL path in stats. Retries count as separate attempts; dry-run and
// mock requests are not counted.
func CountByMethodAndPath(stats *StatsMap) Middleware {
	return func(next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if !isNotSent(r.Context()) && r.URL != nil {
				stats.Inc(r.Method, r.URL.Path)
			}
			return next.RoundTrip(r)
		})
	}
}
