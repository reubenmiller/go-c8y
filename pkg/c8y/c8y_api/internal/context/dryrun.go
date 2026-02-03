package contexthelpers

import (
	"context"
)

// Use a unique unexported key type to avoid collisions with other packages
type dryRunKey struct{}
type redactHeadersKey struct{}
type deferredExecutionKey struct{}

// WithDryRun returns a context with dry run enabled
func WithDryRun(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, dryRunKey{}, enabled)
}

// IsDryRun checks if dry run is enabled in the context
func IsDryRun(ctx context.Context) bool {
	if v, ok := ctx.Value(dryRunKey{}).(bool); ok {
		return v
	}
	return false
}

// WithRedactHeaders returns a context with header redaction control
// By default, headers are redacted for security. Set to false to disable redaction for debugging.
func WithRedactHeaders(ctx context.Context, redact bool) context.Context {
	return context.WithValue(ctx, redactHeadersKey{}, redact)
}

// ShouldRedactHeaders checks if header redaction is enabled in the context
// Returns true by default (secure by default)
func ShouldRedactHeaders(ctx context.Context) bool {
	if v, ok := ctx.Value(redactHeadersKey{}).(bool); ok {
		return v
	}
	return true // Default to redacting for security
}

// WithDeferredExecution returns a context with deferred execution enabled
// When enabled, operations will prepare the request (including parameter resolution)
// but won't execute the HTTP call until Result.Execute() is called.
func WithDeferredExecution(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, deferredExecutionKey{}, enabled)
}

// IsDeferredExecution checks if deferred execution is enabled in the context
func IsDeferredExecution(ctx context.Context) bool {
	if v, ok := ctx.Value(deferredExecutionKey{}).(bool); ok {
		return v
	}
	return false
}
