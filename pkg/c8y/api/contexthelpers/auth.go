package contexthelpers

import "context"

type noAuthKey struct{}

// WithNoAuth returns a context that instructs the transport layer to strip the
// Authorization header before sending, allowing requests to public endpoints
// (e.g. /tenant/loginOptions) that do not accept authentication.
func WithNoAuth(ctx context.Context) context.Context {
	return context.WithValue(ctx, noAuthKey{}, true)
}

// IsNoAuth reports whether the request should be sent without any Authorization header.
func IsNoAuth(ctx context.Context) bool {
	v, _ := ctx.Value(noAuthKey{}).(bool)
	return v
}
