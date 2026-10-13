package contexthelpers

import (
	"context"
	"net/url"
)

type extraHeadersKey struct{}
type extraQueryKey struct{}
type noAcceptKey struct{}

// WithNoAccept returns a context that asks the transport to strip the Accept
// header from the request (the CLI's --noAccept flag). When set, a POST/PUT
// returns the created/updated object without forcing a representation.
func WithNoAccept(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, noAcceptKey{}, enabled)
}

// IsNoAccept reports whether the Accept header should be stripped.
func IsNoAccept(ctx context.Context) bool {
	v, _ := ctx.Value(noAcceptKey{}).(bool)
	return v
}

// WithExtraHeaders returns a context carrying custom request headers that the
// request layer adds to every outgoing request (the CLI's global --header
// setting). An empty map is a no-op.
func WithExtraHeaders(ctx context.Context, headers map[string]string) context.Context {
	if len(headers) == 0 {
		return ctx
	}
	return context.WithValue(ctx, extraHeadersKey{}, headers)
}

// GetExtraHeaders returns the custom request headers from the context, or nil.
func GetExtraHeaders(ctx context.Context) map[string]string {
	if v, ok := ctx.Value(extraHeadersKey{}).(map[string]string); ok {
		return v
	}
	return nil
}

// WithExtraQueryParams returns a context carrying custom query parameters that
// the request layer adds to every outgoing request (the CLI's global
// --customQueryParam setting). An empty value is a no-op.
func WithExtraQueryParams(ctx context.Context, params url.Values) context.Context {
	if len(params) == 0 {
		return ctx
	}
	return context.WithValue(ctx, extraQueryKey{}, params)
}

// GetExtraQueryParams returns the custom query parameters from the context, or nil.
func GetExtraQueryParams(ctx context.Context) url.Values {
	if v, ok := ctx.Value(extraQueryKey{}).(url.Values); ok {
		return v
	}
	return nil
}
