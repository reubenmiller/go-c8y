package contexthelpers

import "context"

type conflictAsErrorKey struct{}

// WithConflictAsError makes idempotent calls keep their 409 error.
func WithConflictAsError(ctx context.Context) context.Context {
	return context.WithValue(ctx, conflictAsErrorKey{}, true)
}

// ConflictAsError reports whether WithConflictAsError is set.
func ConflictAsError(ctx context.Context) bool {
	v, _ := ctx.Value(conflictAsErrorKey{}).(bool)
	return v
}
