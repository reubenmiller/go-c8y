package contexthelpers

import (
	"context"
	"net/http"
)

// Use a unique unexported key type to avoid collisions with other packages
type dryRunKey struct{}
type mockResponsesKey struct{}
type dryRunHandlerKey struct{}
type mockResponsesAutoKey struct{}

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

// WithDryRun returns a context with dry run enabled.
// Dry run mode logs requests for inspection/validation.
//
// For backward compatibility, dry run also enables mock responses unless the
// caller has set them explicitly with WithMockResponses, before or after this
// call: WithMockResponses(WithDryRun(ctx, true), false) and
// WithDryRun(WithMockResponses(ctx, false), true) both give a pure dry run.
func WithDryRun(ctx context.Context, enabled bool) context.Context {
	ctx = context.WithValue(ctx, dryRunKey{}, enabled)
	if enabled {
		if _, explicit := ctx.Value(mockResponsesKey{}).(bool); !explicit || IsMockResponsesAuto(ctx) {
			ctx = context.WithValue(ctx, mockResponsesKey{}, true)
			ctx = context.WithValue(ctx, mockResponsesAutoKey{}, true)
		}
	}
	return ctx
}

// IsMockResponsesAuto reports whether mock responses were enabled implicitly by
// WithDryRun rather than requested explicitly with WithMockResponses.
func IsMockResponsesAuto(ctx context.Context) bool {
	v, _ := ctx.Value(mockResponsesAutoKey{}).(bool)
	return v
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
	ctx = context.WithValue(ctx, mockResponsesKey{}, enabled)
	return context.WithValue(ctx, mockResponsesAutoKey{}, false)
}

// IsMockResponses checks if mock responses are enabled in the context
func IsMockResponses(ctx context.Context) bool {
	if v, ok := ctx.Value(mockResponsesKey{}).(bool); ok {
		return v
	}
	return false
}
