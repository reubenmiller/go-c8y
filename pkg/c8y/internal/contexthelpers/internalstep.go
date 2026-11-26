package contexthelpers

import "context"

// InternalStep returns a context for a step of a composite SDK operation, for
// example the activation or the subscription inside a deploy.
//
// WithNotFoundAsError and WithConflictAsError apply to the operation the
// caller asked for. A composite operation relies on ignoring a 404 or a 409
// in its own steps (an already active version, an already subscribed
// application), so it sends those steps with InternalStep, which turns both
// options off. The returned context is otherwise unchanged.
func InternalStep(ctx context.Context) context.Context {
	if NotFoundAsError(ctx) {
		ctx = context.WithValue(ctx, notFoundAsErrorKey{}, false)
	}
	if ConflictAsError(ctx) {
		ctx = context.WithValue(ctx, conflictAsErrorKey{}, false)
	}
	return ctx
}
