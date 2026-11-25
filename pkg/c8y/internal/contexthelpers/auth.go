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

type explicitAuthTypeKey struct{}

// WithExplicitAuthType marks a request as having its own authentication type, so
// the client-level AuthType does not apply to it.
func WithExplicitAuthType(ctx context.Context) context.Context {
	return context.WithValue(ctx, explicitAuthTypeKey{}, true)
}

// IsExplicitAuthType reports whether the request chose its own authentication type.
func IsExplicitAuthType(ctx context.Context) bool {
	v, _ := ctx.Value(explicitAuthTypeKey{}).(bool)
	return v
}
