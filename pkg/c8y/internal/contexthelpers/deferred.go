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
// Resolution must happen eagerly and reach the real API even when the
// surrounding request is a dry run or is deferred: the lookup HTTP call must
// fire so the resolved ID is available to embed in the request those modes
// report or prepare. ResolutionContext therefore clears the dry-run and
// deferred-execution flags, the mock responses that dry run enabled implicitly,
// and any request approver (lookups are reads and never need approval).
//
// Everything else on ctx is kept: authentication, tenant, extra headers and
// query parameters, processing mode, cancellation and deadlines. A caller that
// has explicitly enabled mock responses keeps them, so tests still work.
//
// Usage:
//
//	resolutionCtx := ctxhelpers.ResolutionContext(ctx)
//	resolvedID, err := resolver.ResolveID(resolutionCtx, source, meta)
func ResolutionContext(ctx context.Context) context.Context {
	if IsDeferredExecution(ctx) {
		ctx = WithDeferredExecution(ctx, false)
	}
	if IsDryRun(ctx) {
		ctx = context.WithValue(ctx, dryRunKey{}, false)
	}
	if IsMockResponsesAuto(ctx) {
		ctx = WithMockResponses(ctx, false)
	}
	if GetRequestApprover(ctx) != nil {
		ctx = WithRequestApprover(ctx, nil)
	}
	return ctx
}
