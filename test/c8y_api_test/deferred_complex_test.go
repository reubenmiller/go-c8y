package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_DeferredExecution_GetOrCreateByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	// Prepare a get-or-create operation
	prepared := client.ManagedObjects.GetOrCreateByName(ctx, "test-device", "c8y_Device", map[string]any{
		"name": "test-device",
		"type": "c8y_Device",
	})

	// Should be deferred
	assert.True(t, prepared.IsDeferred(), "GetOrCreateByName should support deferred execution")

	// Metadata should indicate operation type
	assert.Equal(t, "getOrCreateByName", prepared.Meta["operation"])

	// Note: We don't execute it in this test to avoid creating test data
}

func Test_DeferredExecution_GetOrCreateByFragment(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	// Prepare a get-or-create operation
	prepared := client.ManagedObjects.GetOrCreateByFragment(ctx, "c8y_IsDevice", map[string]any{
		"name":         "test-device",
		"c8y_IsDevice": map[string]any{},
	})

	// Should be deferred
	assert.True(t, prepared.IsDeferred(), "GetOrCreateByFragment should support deferred execution")

	// Metadata should indicate operation type
	assert.Equal(t, "getOrCreateByFragment", prepared.Meta["operation"])
}

func Test_DeferredExecution_GetOrCreateWith(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	// Prepare a get-or-create operation
	query := "name eq 'test-device' and type eq 'c8y_Device'"
	prepared := client.ManagedObjects.GetOrCreateWith(ctx, map[string]any{
		"name": "test-device",
		"type": "c8y_Device",
	}, query)

	// Should be deferred
	assert.True(t, prepared.IsDeferred(), "GetOrCreateWith should support deferred execution")

	// Metadata should indicate operation type
	assert.Equal(t, "getOrCreateWith", prepared.Meta["operation"])
}

func Test_DeferredExecution_GetOrCreateByExternalID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)

	// Prepare a get-or-create operation
	prepared := client.ManagedObjects.GetOrCreateByExternalID(ctx, managedobjects.GetOrCreateByExternalIDOptions{
		ExternalID:     "SERIAL-12345",
		ExternalIDType: "c8y_Serial",
		Body: map[string]any{
			"name": "test-device",
			"type": "c8y_Device",
		},
	})

	// Should be deferred
	assert.True(t, prepared.IsDeferred(), "GetOrCreateByExternalID should support deferred execution")

	// Metadata should indicate operation type
	assert.Equal(t, "getOrCreateByExternalID", prepared.Meta["operation"])
}
