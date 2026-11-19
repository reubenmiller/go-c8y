package contexthelpers

import "context"

type skipContextTokenExchangeKey struct{}

// WithSkipContextTokenExchange returns a context that instructs the
// context-authorization middleware to use the per-request basic credentials
// directly instead of exchanging them for a bearer token. Used internally to
// retry a request with basic auth after an exchanged token was rejected.
func WithSkipContextTokenExchange(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipContextTokenExchangeKey{}, true)
}

// IsSkipContextTokenExchange reports whether the context token exchange should
// be skipped for this request.
func IsSkipContextTokenExchange(ctx context.Context) bool {
	v, _ := ctx.Value(skipContextTokenExchangeKey{}).(bool)
	return v
}
