package context

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/stretchr/testify/assert"
)

func TestProcessingModeContext(t *testing.T) {
	tests := []struct {
		name         string
		setupContext func(ctx context.Context) context.Context
		expected     types.ProcessingMode
	}{
		{
			name:         "WithProcessingModePersistent",
			setupContext: WithProcessingModePersistent,
			expected:     types.ProcessingModePersistent,
		},
		{
			name:         "WithProcessingModeTransient",
			setupContext: WithProcessingModeTransient,
			expected:     types.ProcessingModeTransient,
		},
		{
			name:         "WithProcessingModeQuiescent",
			setupContext: WithProcessingModeQuiescent,
			expected:     types.ProcessingModeQuiescent,
		},
		{
			name:         "WithProcessingModeCEP",
			setupContext: WithProcessingModeCEP,
			expected:     types.ProcessingModeCEP,
		},
		{
			name: "WithProcessingMode generic",
			setupContext: func(ctx context.Context) context.Context {
				return WithProcessingMode(ctx, types.ProcessingModeTransient)
			},
			expected: types.ProcessingModeTransient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = tt.setupContext(ctx)

			actual := GetProcessingMode(ctx)
			assert.Equal(t, tt.expected, actual, "GetProcessingMode() should return expected mode")
		})
	}
}

func TestProcessingModeNotSet(t *testing.T) {
	ctx := context.Background()
	mode := GetProcessingMode(ctx)

	assert.Empty(t, mode, "GetProcessingMode() should return empty string when not set")
}

func TestProcessingModeOverride(t *testing.T) {
	ctx := context.Background()

	// Set to persistent first
	ctx = WithProcessingModePersistent(ctx)
	mode := GetProcessingMode(ctx)
	assert.Equal(t, types.ProcessingModePersistent, mode, "Should set to persistent mode")

	// Override with transient
	ctx = WithProcessingModeTransient(ctx)
	mode = GetProcessingMode(ctx)
	assert.Equal(t, types.ProcessingModeTransient, mode, "Should override to transient mode")
}
