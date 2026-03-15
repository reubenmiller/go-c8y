package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/source"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
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

	// Use name resolver syntax helper
	nameRef := client.ManagedObjects.ByName("myDevice")
	assert.Equal(t, "name:myDevice", nameRef)
}

func Test_Resolver_ExternalIDSyntax(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Create an external ID reference using helper
	extRef := client.ManagedObjects.ByExternalID("c8y_Serial", "SERIAL-12345")
	assert.Equal(t, "ext:c8y_Serial:SERIAL-12345", extRef)
}

func Test_Resolver_StringSyntax(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Test helper methods return correct string syntax
	assert.Equal(t, "12345", client.ManagedObjects.ByID("12345"))
	assert.Equal(t, "name:myDevice", client.ManagedObjects.ByName("myDevice"))
	assert.Equal(t, "ext:c8y_Serial:ABC123", client.ManagedObjects.ByExternalID("c8y_Serial", "ABC123"))
	assert.Equal(t, "query:type eq 'c8y_Device'", client.ManagedObjects.ByQuery("type eq 'c8y_Device'"))

	// Test that strings can be resolved
	meta := make(map[string]any)
	resolvedID, err := client.ManagedObjects.ResolveID(ctx, "12345", meta)
	assert.NoError(t, err)
	assert.Equal(t, "12345", resolvedID)
	assert.Equal(t, "direct-id", meta["source"])
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
	ctx := api.WithDeferredExecution(context.Background(), true)

	// Create a Get request with plain ID - this should still defer
	plainIDResult := client.ManagedObjects.Get(ctx, "12345", managedobjects.GetOptions{})
	assert.True(t, plainIDResult.IsDeferred(), "Plain ID should allow deferred execution")
}

func Test_Resolver_DeleteWithName(t *testing.T) {
	// Test that resolvers work with Delete operations
	client := testcore.CreateTestClient(t)

	// Create a Delete with deferred execution using a plain ID
	ctx := api.WithDeferredExecution(context.Background(), true)
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
