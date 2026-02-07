package context

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	contexthelpers "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/internal/context"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/stretchr/testify/assert"
	"resty.dev/v3"
)

func TestProcessingModeIntegration(t *testing.T) {
	// Create a test client
	client := resty.New()

	tests := []struct {
		name           string
		setupContext   func(ctx context.Context) context.Context
		expectedHeader string
	}{
		{
			name:           "PERSISTENT mode sets header",
			setupContext:   WithProcessingModePersistent,
			expectedHeader: "PERSISTENT",
		},
		{
			name:           "TRANSIENT mode sets header",
			setupContext:   WithProcessingModeTransient,
			expectedHeader: "TRANSIENT",
		},
		{
			name:           "QUIESCENT mode sets header",
			setupContext:   WithProcessingModeQuiescent,
			expectedHeader: "QUIESCENT",
		},
		{
			name:           "CEP mode sets header",
			setupContext:   WithProcessingModeCEP,
			expectedHeader: "CEP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup context with dry run to avoid actual network calls
			ctx := context.Background()
			ctx = contexthelpers.WithDryRun(ctx, true)
			ctx = tt.setupContext(ctx)

			// Create a request
			req := client.R().SetURL("https://example.c8y.com/inventory/managedObjects")
			tryReq := core.NewTryRequest(client, req)

			// Apply context which should set the processing mode header
			tryReq.SetContext(ctx)

			// Check that the header was set correctly
			headerValue := req.Header.Get(types.HeaderProcessingMode)
			assert.Equal(t, tt.expectedHeader, headerValue, "Processing mode header should be set correctly")
		})
	}
}

func TestProcessingModeNotSetIntegration(t *testing.T) {
	// Test that no header is set when no processing mode is in context
	client := resty.New()
	ctx := context.Background()
	ctx = contexthelpers.WithDryRun(ctx, true)

	req := client.R().SetURL("https://example.c8y.com/inventory/managedObjects")
	tryReq := core.NewTryRequest(client, req)

	tryReq.SetContext(ctx)

	// Check that no processing mode header is set
	headerValue := req.Header.Get(types.HeaderProcessingMode)
	assert.Empty(t, headerValue, "No processing mode header should be set when not in context")
}
