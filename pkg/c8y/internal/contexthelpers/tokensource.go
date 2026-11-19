package contexthelpers

import "context"

type skipTokenSourceKey struct{}

// WithSkipTokenSource returns a context that instructs the TokenSource
// middleware to skip token injection. Used internally so that credential-fetch
// requests (e.g. the internal username/password login call) do not re-trigger
// the middleware and cause infinite recursion.
func WithSkipTokenSource(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipTokenSourceKey{}, true)
}

// IsSkipTokenSource reports whether the TokenSource middleware should be skipped.
func IsSkipTokenSource(ctx context.Context) bool {
	v, _ := ctx.Value(skipTokenSourceKey{}).(bool)
	return v
}
