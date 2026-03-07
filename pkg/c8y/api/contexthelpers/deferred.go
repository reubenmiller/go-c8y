package contexthelpers

import (
	"context"
)

type deferredExecutionKey struct{}

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

// ResolutionContext returns a context suitable for resolving device/resource IDs.
//
// Device resolution must happen eagerly even in deferred-execution mode (the
// lookup HTTP call must actually fire so we have the resolved ID to embed in the
// deferred request). ResolutionContext strips the deferred-execution flag and
// preserves any mock-response overrides so tests continue to work correctly.
//
// Usage:
//
//	resolutionCtx := ctxhelpers.ResolutionContext(ctx)
//	resolvedID, err := resolver.ResolveID(resolutionCtx, source, meta)
func ResolutionContext(ctx context.Context) context.Context {
	if !IsDeferredExecution(ctx) {
		return ctx
	}
	return WithMockResponses(context.Background(), IsMockResponses(ctx))
}
