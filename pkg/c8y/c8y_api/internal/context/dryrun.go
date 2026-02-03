package contexthelpers

import (
	"context"
)

// Use a unique unexported key type to avoid collisions with other packages
type dryRunKey struct{}
type redactHeadersKey struct{}

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
