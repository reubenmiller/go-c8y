package contexthelpers

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
)

type processingModeKey struct{}

// WithProcessingMode returns a context with the specified Cumulocity processing mode
func WithProcessingMode(ctx context.Context, mode types.ProcessingMode) context.Context {
	return context.WithValue(ctx, processingModeKey{}, mode)
}

// WithProcessingModePersistent sets the processing mode to PERSISTENT (default mode)
// Stores data in the Cumulocity database and sends data to the Streaming Analytics engine
func WithProcessingModePersistent(ctx context.Context) context.Context {
	return WithProcessingMode(ctx, types.ProcessingModePersistent)
}

// WithProcessingModeTransient sets the processing mode to TRANSIENT
// Sends data to the Streaming Analytics engine but does not store data in Cumulocity's database
func WithProcessingModeTransient(ctx context.Context) context.Context {
	return WithProcessingMode(ctx, types.ProcessingModeTransient)
}

// WithProcessingModeQuiescent sets the processing mode to QUIESCENT
// Similar to persistent mode but no real-time notifications will be sent
func WithProcessingModeQuiescent(ctx context.Context) context.Context {
	return WithProcessingMode(ctx, types.ProcessingModeQuiescent)
}

// WithProcessingModeCEP sets the processing mode to CEP
// Like transient mode but no real-time notifications are sent
func WithProcessingModeCEP(ctx context.Context) context.Context {
	return WithProcessingMode(ctx, types.ProcessingModeCEP)
}

// GetProcessingMode retrieves the processing mode from the context
// Returns empty string if no processing mode is set
func GetProcessingMode(ctx context.Context) types.ProcessingMode {
	if v, ok := ctx.Value(processingModeKey{}).(types.ProcessingMode); ok {
		return v
	}
	return ""
}
