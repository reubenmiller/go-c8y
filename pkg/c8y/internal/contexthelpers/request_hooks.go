package contexthelpers

import (
	"context"
	"io"
	"net/http"
	"sync"
)

type requestHookKey struct{}

// WithRequestHook returns a context whose requests are passed to fn just before
// they are sent (once per attempt). An error from fn aborts the request.
func WithRequestHook(ctx context.Context, fn func(*http.Request) error) context.Context {
	return context.WithValue(ctx, requestHookKey{}, fn)
}

// RequestHook returns the hook set with WithRequestHook, or nil.
func RequestHook(ctx context.Context) func(*http.Request) error {
	fn, _ := ctx.Value(requestHookKey{}).(func(*http.Request) error)
	return fn
}

// RequestCapture receives a copy of the last request sent with a context
// created by WithRequestCapture. The copy owns its body, so it stays valid
// after the HTTP client reuses its own buffers.
type RequestCapture struct {
	mu      sync.Mutex
	request *http.Request
}

// Set stores the captured request.
func (c *RequestCapture) Set(r *http.Request) {
	c.mu.Lock()
	c.request = r
	c.mu.Unlock()
}

// Request returns the captured request, or nil when nothing was sent.
func (c *RequestCapture) Request() *http.Request {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.request
}

type requestCaptureKey struct{}

// WithRequestCapture returns a context whose sent requests are copied into c.
func WithRequestCapture(ctx context.Context, c *RequestCapture) context.Context {
	return context.WithValue(ctx, requestCaptureKey{}, c)
}

// GetRequestCapture returns the capture set with WithRequestCapture, or nil.
func GetRequestCapture(ctx context.Context) *RequestCapture {
	c, _ := ctx.Value(requestCaptureKey{}).(*RequestCapture)
	return c
}

type responseBodyWrapperKey struct{}

// WithResponseBodyWrapper returns a context whose responses have their body
// replaced by fn(resp) once the response headers have arrived.
func WithResponseBodyWrapper(ctx context.Context, fn func(*http.Response) io.ReadCloser) context.Context {
	return context.WithValue(ctx, responseBodyWrapperKey{}, fn)
}

// ResponseBodyWrapper returns the wrapper set with WithResponseBodyWrapper, or nil.
func ResponseBodyWrapper(ctx context.Context) func(*http.Response) io.ReadCloser {
	fn, _ := ctx.Value(responseBodyWrapperKey{}).(func(*http.Response) io.ReadCloser)
	return fn
}
