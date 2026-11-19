package contexthelpers

import (
	"context"
)

type redactHeadersKey struct{}

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
