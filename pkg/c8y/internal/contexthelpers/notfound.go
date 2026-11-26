package contexthelpers

import "context"

type notFoundAsErrorKey struct{}

// WithNotFoundAsError makes idempotent calls keep their 404 error.
func WithNotFoundAsError(ctx context.Context) context.Context {
	return context.WithValue(ctx, notFoundAsErrorKey{}, true)
}

// NotFoundAsError reports whether WithNotFoundAsError is set.
func NotFoundAsError(ctx context.Context) bool {
	v, _ := ctx.Value(notFoundAsErrorKey{}).(bool)
	return v
}
