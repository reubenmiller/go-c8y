package contexthelpers

import (
	"context"
	"net/http"
)

// Use a unique unexported key type to avoid collisions with other packages
type dryRunKey struct{}
type mockResponsesKey struct{}
type dryRunHandlerKey struct{}

// WithDryRunHandler registers a callback that receives the prepared request in
// dry-run mode, letting a caller (e.g. a CLI) render it in its own format
// instead of the default slog line. The request is not sent.
func WithDryRunHandler(ctx context.Context, fn func(*http.Request)) context.Context {
	return context.WithValue(ctx, dryRunHandlerKey{}, fn)
}

// DryRunHandler returns the registered dry-run handler, or nil.
func DryRunHandler(ctx context.Context) func(*http.Request) {
	if fn, ok := ctx.Value(dryRunHandlerKey{}).(func(*http.Request)); ok {
		return fn
	}
	return nil
}

// WithDryRun returns a context with dry run enabled
// Dry run mode logs requests for inspection/validation
// For backward compatibility, this also enables mock responses by default
// To have dry run without mocks, use WithDryRun followed by WithMockResponses(ctx, false)
func WithDryRun(ctx context.Context, enabled bool) context.Context {
	ctx = context.WithValue(ctx, dryRunKey{}, enabled)
	// Backward compatibility: enable mock responses by default with dry run
	if enabled && !IsMockResponses(ctx) {
		ctx = WithMockResponses(ctx, true)
	}
	return ctx
}

// IsDryRun checks if dry run is enabled in the context
func IsDryRun(ctx context.Context) bool {
	if v, ok := ctx.Value(dryRunKey{}).(bool); ok {
		return v
	}
	return false
}

// WithMockResponses returns a context with mock responses enabled
// When enabled, HTTP requests will return mock data from embedded JSON files
// instead of making real API calls. Useful for unit testing without network dependencies.
func WithMockResponses(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, mockResponsesKey{}, enabled)
}

// IsMockResponses checks if mock responses are enabled in the context
func IsMockResponses(ctx context.Context) bool {
	if v, ok := ctx.Value(mockResponsesKey{}).(bool); ok {
		return v
	}
	return false
}
