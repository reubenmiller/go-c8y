package context

import (
	"context"

	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/api/internal/context"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
)

// WithProcessingMode returns a context with the specified Cumulocity processing mode
// When this context is used with API operations, the X-Cumulocity-Processing-Mode header will be set
func WithProcessingMode(ctx context.Context, mode types.ProcessingMode) context.Context {
	return ctxhelpers.WithProcessingMode(ctx, mode)
}

// WithProcessingModePersistent sets the processing mode to PERSISTENT (default mode)
// Stores data in the Cumulocity database and sends data to the Streaming Analytics engine.
// Afterwards, Cumulocity returns the result of the request. This is the default mode.
func WithProcessingModePersistent(ctx context.Context) context.Context {
	return ctxhelpers.WithProcessingModePersistent(ctx)
}

// WithProcessingModeTransient sets the processing mode to TRANSIENT
// Sends data to the Streaming Analytics engine and immediately returns the results asynchronously
// but does not store data in Cumulocity's database. This mode saves storage and processing costs
// and is useful for example when tracking devices in real time without requiring data to be stored.
func WithProcessingModeTransient(ctx context.Context) context.Context {
	return ctxhelpers.WithProcessingModeTransient(ctx)
}

// WithProcessingModeQuiescent sets the processing mode to QUIESCENT
// Behaves similar to the persistent mode with the exception that no real-time notifications will be sent.
// The quiescent processing mode is applicable only for measurements and events.
func WithProcessingModeQuiescent(ctx context.Context) context.Context {
	return ctxhelpers.WithProcessingModeQuiescent(ctx)
}

// WithProcessingModeCEP sets the processing mode to CEP
// Behaves like the transient mode with the exception that no real-time notifications are sent.
// Currently it is applicable only for measurements and events.
func WithProcessingModeCEP(ctx context.Context) context.Context {
	return ctxhelpers.WithProcessingModeCEP(ctx)
}

// GetProcessingMode retrieves the processing mode from the context
// Returns empty string if no processing mode is set
func GetProcessingMode(ctx context.Context) types.ProcessingMode {
	return ctxhelpers.GetProcessingMode(ctx)
}
