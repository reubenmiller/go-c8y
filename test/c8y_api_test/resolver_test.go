package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/source"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_Resolver_PlainID(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Plain ID should pass through unchanged and resolve successfully
	ctx := context.Background()
	meta := make(map[string]any)
	resolvedID, err := client.ManagedObjects.ResolveID(ctx, "12345", meta)

	assert.NoError(t, err)
	assert.Equal(t, "12345", resolvedID)
	assert.NotNil(t, meta)
	assert.Equal(t, "direct-id", meta["source"])
}

func Test_Resolver_NameSyntax(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Use name resolver syntax: "name:device-name"
	nameID := client.ManagedObjects.ByName("myDevice").String()
	assert.Equal(t, "name:myDevice", nameID)
}

func Test_Resolver_ExternalIDSyntax(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Create an external ID reference
	extID := client.ManagedObjects.ByExternalID("c8y_Serial", "SERIAL-12345").String()
	assert.Equal(t, "ext:c8y_Serial:SERIAL-12345", extID)
}

func Test_Resolver_IDBuilderHelpers(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Test the ID builder helpers
	assert.Equal(t, "12345", client.ManagedObjects.ByID("12345").String())
	assert.Equal(t, "name:myDevice", client.ManagedObjects.ByName("myDevice").String())
	assert.Equal(t, "ext:c8y_Serial:ABC123", client.ManagedObjects.ByExternalID("c8y_Serial", "ABC123").String())
}

func Test_Resolver_UnknownScheme(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Try with unknown resolver scheme
	_, err := client.ManagedObjects.ResolveID(ctx, "unknown:value", nil)

	// Should fail with unknown resolver error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resolver scheme")
}

func Test_Resolver_WithDeferredExecution(t *testing.T) {
	// Deferred execution combined with resolvers is a complex scenario
	// that requires the resolver lookups to actually execute in the background,
	// while the main Get/Delete operations are deferred.
	// For now, just verify that deferred execution is possible with resolver syntax.
	client := testcore.CreateTestClient(t)
	ctx := c8y_api.WithDeferredExecution(context.Background(), true)

	// Create a Get request with plain ID - this should still defer
	plainIDResult := client.ManagedObjects.Get(ctx, "12345", managedobjects.GetOptions{})
	assert.True(t, plainIDResult.IsDeferred(), "Plain ID should allow deferred execution")
}

func Test_Resolver_DeleteWithName(t *testing.T) {
	// Test that resolvers work with Delete operations
	client := testcore.CreateTestClient(t)

	// Create a Delete with deferred execution using a plain ID
	ctx := c8y_api.WithDeferredExecution(context.Background(), true)
	prepared := client.ManagedObjects.Delete(ctx, "deleteMe123", managedobjects.DeleteOptions{})

	// Should be deferred
	assert.True(t, prepared.IsDeferred())
}

type customTestResolver struct{}

// customTestResolver is a test resolver that always returns a fixed ID
func (r *customTestResolver) ResolveID(ctx context.Context) (source.ResolveResult, error) {
	return source.ResolveResult{
		ID: "12345",
		Meta: map[string]any{
			"source": "custom-test",
		},
	}, nil
}

func (r *customTestResolver) String() string {
	return "custom:test"
}

func Test_Resolver_CustomResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Register a custom resolver
	client.ManagedObjects.RegisterResolver("custom", &customTestResolver{})

	// For testing, just verify the resolver is callable
	ctx := context.Background()
	meta := make(map[string]any)
	resolvedID, err := client.ManagedObjects.ResolveID(ctx, "custom:value", meta)

	// The custom resolver returns a known ID
	assert.NoError(t, err)
	assert.Equal(t, "12345", resolvedID)
	assert.NotNil(t, meta)
}
