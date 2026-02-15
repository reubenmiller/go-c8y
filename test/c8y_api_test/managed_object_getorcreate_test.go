package api_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_ManagedObjectGetOrCreateByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)
	ctx := context.Background()

	name := testingutils.RandomString(16)
	objType := "test_device_getorcreate"
	body := map[string]any{
		"name":         name,
		"type":         objType,
		"c8y_IsDevice": map[string]any{},
	}

	// First call - should create
	result1 := client.ManagedObjects.GetOrCreateByName(ctx, name, objType, body)
	assert.NoError(t, result1.Err)
	assert.Equal(t, "Created", string(result1.Status))
	assert.Equal(t, name, result1.Data.Name())
	assert.Equal(t, objType, result1.Data.Type())
	assert.False(t, result1.Meta["found"].(bool))

	id := result1.Data.ID()

	// Second call with same name/type - should find existing
	result2 := client.ManagedObjects.GetOrCreateByName(ctx, name, objType, body)
	assert.NoError(t, result2.Err)
	assert.Equal(t, "OK", string(result2.Status))
	assert.Equal(t, id, result2.Data.ID())
	assert.Equal(t, name, result2.Data.Name())
	assert.True(t, result2.Meta["found"].(bool))

	// Cleanup
	deleteResult := client.ManagedObjects.Delete(ctx, id, managedobjects.DeleteOptions{})
	assert.NoError(t, deleteResult.Err)
}

func Test_ManagedObjectGetOrCreateByFragment(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := testingutils.RandomString(16)
	uniqueFragment := "c8y_CustomIdentifier_" + name
	body := map[string]any{
		"name":         name,
		"type":         "test_device_fragment",
		uniqueFragment: map[string]any{},
	}

	// First call - should create
	result1 := client.ManagedObjects.GetOrCreateByFragment(ctx, uniqueFragment, body)
	assert.NoError(t, result1.Err)
	assert.Equal(t, "Created", string(result1.Status))
	assert.False(t, result1.Meta["found"].(bool))

	id := result1.Data.ID()

	// Second call - should find by fragment
	result2 := client.ManagedObjects.GetOrCreateByFragment(ctx, uniqueFragment, body)
	assert.NoError(t, result2.Err)
	assert.Equal(t, "OK", string(result2.Status))
	assert.Equal(t, id, result2.Data.ID())
	assert.True(t, result2.Meta["found"].(bool))

	// Cleanup
	deleteResult := client.ManagedObjects.Delete(ctx, id, managedobjects.DeleteOptions{})
	assert.NoError(t, deleteResult.Err)
}

func Test_ManagedObjectGetOrCreateWith(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	serialNumber := testingutils.RandomString(12)
	body := map[string]any{
		"name": testingutils.RandomString(16),
		"type": "test_device_serial",
		"c8y_Hardware": map[string]any{
			"serialNumber": serialNumber,
		},
	}

	// Custom query for serial number
	query := fmt.Sprintf("has(c8y_Hardware) and c8y_Hardware.serialNumber eq '%s'", serialNumber)

	// First call - should create
	result1 := client.ManagedObjects.GetOrCreateWith(ctx, body, query)
	assert.NoError(t, result1.Err)
	assert.Equal(t, "Created", string(result1.Status))
	assert.False(t, result1.Meta["found"].(bool))

	id := result1.Data.ID()

	// Second call with same query - should find
	result2 := client.ManagedObjects.GetOrCreateWith(ctx, body, query)
	assert.NoError(t, result2.Err)
	assert.Equal(t, "OK", string(result2.Status))
	assert.Equal(t, id, result2.Data.ID())
	assert.True(t, result2.Meta["found"].(bool))

	// Cleanup
	deleteResult := client.ManagedObjects.Delete(ctx, id, managedobjects.DeleteOptions{})
	assert.NoError(t, deleteResult.Err)
}
