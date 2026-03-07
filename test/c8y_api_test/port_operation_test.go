package api_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/operations"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CreateOperation(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	device := testcore.CreateDeviceAgent(t, client).Data

	// Create operation
	body := map[string]any{
		"deviceId": device.ID(),
		"test_operation": map[string]any{
			"name": "test operation 1",
			"parameters": map[string]any{
				"value1": 1,
			},
		},
	}

	result := client.Operations.Create(ctx, body)
	require.NoError(t, result.Err)
	assert.Equal(t, 201, result.HTTPStatus)
	assert.Equal(t, device.ID(), result.Data.DeviceID())

	// Get operations list
	listResult := client.Operations.List(ctx, operations.ListOptions{
		DeviceID: managedobjects.DeviceRef(device.ID()),
	})

	require.NoError(t, listResult.Err)
	assert.Equal(t, 200, listResult.HTTPStatus)

	// Convert to slice to check count
	opsList, err := op.ToSliceR(listResult)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(opsList), 1, "Should have at least one operation")

	// Check that the created operation is in the list
	assert.Equal(t, result.Data.ID(), opsList[0].ID())
}

func Test_UpdateOperation(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	device := testcore.CreateDeviceAgent(t, client).Data

	// Create operation
	body := map[string]any{
		"deviceId": device.ID(),
		"test_operation": map[string]any{
			"name": "test operation for update",
			"parameters": map[string]any{
				"value1": 1,
			},
		},
	}

	op1 := client.Operations.Create(ctx, body)
	require.NoError(t, op1.Err)
	assert.Equal(t, 201, op1.HTTPStatus)

	// Update operation status to EXECUTING
	updateBody1 := map[string]any{
		"status": "EXECUTING",
	}

	op2 := client.Operations.Update(ctx, op1.Data.ID(), updateBody1)
	require.NoError(t, op2.Err)
	assert.Equal(t, 200, op2.HTTPStatus)
	assert.Equal(t, op1.Data.ID(), op2.Data.ID())
	assert.Equal(t, "EXECUTING", op2.Data.Status())

	// Update operation status to FAILED with failure reason
	updateBody2 := map[string]any{
		"status":        "FAILED",
		"failureReason": "Got bored of waiting",
	}

	op3 := client.Operations.Update(ctx, op1.Data.ID(), updateBody2)
	require.NoError(t, op3.Err)
	assert.Equal(t, 200, op3.HTTPStatus)
	assert.Equal(t, op2.Data.ID(), op3.Data.ID())
	assert.Equal(t, "FAILED", op3.Data.Status())
	assert.Equal(t, "Got bored of waiting", op3.Data.Get("failureReason").String())
}

func Test_DeleteOperations(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	device := testcore.CreateDeviceAgent(t, client).Data

	// Create dummy operations
	for i := 1; i <= 3; i++ {
		body := map[string]any{
			"deviceId": device.ID(),
			"test_operation": map[string]any{
				"name": fmt.Sprintf("test operation %d", i),
				"parameters": map[string]any{
					"value1": i,
				},
			},
		}
		result := client.Operations.Create(ctx, body)
		require.NoError(t, result.Err)
		assert.Equal(t, 201, result.HTTPStatus)
	}

	// Check if operations were created
	listResult := client.Operations.List(ctx, operations.ListOptions{
		DeviceID: managedobjects.DeviceRef(device.ID()),
		Status:   "PENDING",
	})

	require.NoError(t, listResult.Err)
	assert.Equal(t, 200, listResult.HTTPStatus)

	opsList, err := op.ToSliceR(listResult)
	require.NoError(t, err)
	assert.Equal(t, 3, len(opsList), "Should have exactly 3 operations")

	// Remove the operations using the same query
	deleteResult := client.Operations.DeleteList(ctx, operations.DeleteListOptions{
		DeviceID: managedobjects.DeviceRef(device.ID()),
		Status:   "PENDING",
	})
	require.NoError(t, deleteResult.Err)
	assert.Equal(t, 204, deleteResult.HTTPStatus)

	// Check count of operations after the delete action
	listResultAfter := client.Operations.List(ctx, operations.ListOptions{
		DeviceID: managedobjects.DeviceRef(device.ID()),
		Status:   "PENDING",
	})

	require.NoError(t, listResultAfter.Err)
	assert.Equal(t, 200, listResultAfter.HTTPStatus)

	opsListAfter, err := op.ToSliceR(listResultAfter)
	require.NoError(t, err)
	assert.Equal(t, 0, len(opsListAfter), "Should have 0 operations after deletion")
}
