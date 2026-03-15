// Package context provides context helpers for configuring Cumulocity API behavior.
//
// # Processing Mode Context Helpers
//
// This package provides convenient context helpers for setting Cumulocity processing modes.
// When a processing mode is set in the context, it will automatically be applied to all
// API requests made with that context by setting the "X-Cumulocity-Processing-Mode" header.
//
// # Available Processing Modes
//
//   - PERSISTENT (default): Stores data in the Cumulocity database and sends data to the Streaming Analytics engine
//   - TRANSIENT: Sends data to the Streaming Analytics engine but does not store data in the database
//   - QUIESCENT: Similar to persistent mode but no real-time notifications will be sent
//   - CEP: Like transient mode but no real-time notifications are sent
//
// # Example Usage
//
//	import (
//	    "context"
//	    c8yctx "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/context"
//	)
//
//	// Set processing mode to transient for real-time tracking without storage
//	ctx := c8yctx.WithProcessingModeTransient(context.Background())
//
//	// All API calls with this context will use TRANSIENT mode
//	result := client.Events.Create(ctx, events.CreateOptions{
//	    Type: "LocationUpdate",
//	    Text: "Device moved",
//	    Source: "12345",
//	})
//
//	// You can also use the generic function with a specific mode
//	ctx = c8yctx.WithProcessingMode(context.Background(), types.ProcessingModeQuiescent)
package context
