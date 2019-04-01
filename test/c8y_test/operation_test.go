package c8y_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
)

func operationFactory(client *c8y.Client, deviceID string) func() (*c8y.Operation, *c8y.Response, error) {
	counter := 1
	return func() (*c8y.Operation, *c8y.Response, error) {
		counter++
		return client.Operation.Create(
			context.Background(),
			map[string]interface{}{
				"deviceId": deviceID,
				"test_operation": map[string]interface{}{
					"name": fmt.Sprintf("test operation %d", counter),
					"parameters": map[string]interface{}{
						"value1": 1,
					},
				},
			},
		)
	}
}

func TestOperationService_CreateOperation(t *testing.T) {
	client := createTestClient()
	device, err := createRandomTestDevice()

	createOp := operationFactory(client, device.ID)
	op, resp, err := createOp()

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode)
	testingutils.Equals(t, device.ID, op.DeviceID)

	ops, resp, err := client.Operation.GetOperations(
		context.Background(),
		&c8y.OperationCollectionOptions{
			DeviceID: device.ID,
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, 1, len(ops.Operations))
	testingutils.Equals(t, 1, len(ops.Items))

	testingutils.Equals(t, op.ID, ops.Operations[0].ID)
	testingutils.Equals(t, op.ID, ops.Items[0].Get("id").String())
}

func TestOperationService_UpdateOperation(t *testing.T) {
	client := createTestClient()
	device, err := createRandomTestDevice()

	createOp := operationFactory(client, device.ID)
	op1, resp, err := createOp()

	testingutils.Ok(t, err)

	op2, resp, err := client.Operation.Update(
		context.Background(),
		op1.ID,
		&c8y.OperationUpdateOptions{
			Status: "EXECUTING",
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, op1.ID, op2.ID)
	testingutils.Equals(t, op2.ID, op2.Item.Get("id").String())

	//
	// Set operation to failed state
	//
	op3, resp, err := client.Operation.Update(
		context.Background(),
		op1.ID,
		&c8y.OperationUpdateOptions{
			Status:        "FAILED",
			FailureReason: "Got bored of waiting",
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, op2.ID, op3.ID)
	testingutils.Equals(t, "Got bored of waiting", op3.Item.Get("failureReason").String())
}

func TestOperationService_DeleteOperation(t *testing.T) {
	client := createTestClient()
	device, err := createRandomTestDevice()

	//
	// Create dummy operations
	//
	createOp := operationFactory(client, device.ID)
	createOp()
	createOp()
	createOp()

	filterOptions := &c8y.OperationCollectionOptions{
		DeviceID: device.ID,
		Status:   "PENDING",
	}

	//
	// Check if operations were created
	//
	ops, resp, err := client.Operation.GetOperations(
		context.Background(),
		filterOptions,
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, 3, len(ops.Operations))
	testingutils.Equals(t, 3, len(ops.Items))

	//
	// Remove the operations using the same query
	//
	resp, err = client.Operation.DeleteOperations(
		context.Background(),
		filterOptions,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode)

	//
	// Check count of operations after the delete action
	//
	ops, resp, err = client.Operation.GetOperations(
		context.Background(),
		filterOptions,
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, 0, len(ops.Operations))
	testingutils.Equals(t, 0, len(ops.Items))
}
