package c8y_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func TestOperationBuilder_CreateOperationBuilder(t *testing.T) {
	id := "12345"
	wantJSON := `{"deviceId":"12345"}`

	builder := c8y.NewOperationBuilder(id)

	operationJSON, err := builder.MarshalJSON()
	testingutils.Ok(t, err)
	testingutils.Equals(t, wantJSON, string(operationJSON))
}

func TestOperationBuilder_CreateOperation(t *testing.T) {
	client := createTestClient()
	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)
	builder := c8y.NewOperationBuilder(testDevice.ID)
	builder.Set("abc_testOperation", map[string]string{
		"parameters": "one",
	})

	jsonStr, err := builder.MarshalJSON()
	testingutils.Ok(t, err)
	testingutils.Assert(t, string(jsonStr) != "", "operation json should be valid")

	op, resp, err := client.Operation.Create(
		context.Background(),
		builder,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Equals(t, testDevice.ID, op.DeviceID)
}

func TestOperationBuilder_CreateAgentUpdateConfigurationOperation(t *testing.T) {
	client := createTestClient()
	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)
	agentConfig := `
testProp=1
	`
	configOp := c8y.NewOperationAgentUpdateConfiguration(testDevice.ID, agentConfig)
	op, resp, err := client.Operation.Create(
		context.Background(),
		configOp,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Equals(t, testDevice.ID, op.DeviceID)
}

func TestOperationBuilder_DeviceID(t *testing.T) {
	builder := c8y.NewOperationBuilder("12345")
	testingutils.Equals(t, "12345", builder.DeviceID())

	builder.SetDeviceID("99192")
	testingutils.Equals(t, "99192", builder.DeviceID())
}

func TestOperationBuilder_GetSet(t *testing.T) {
	builder := c8y.NewOperationBuilder("12345")

	builder.Set("c8y_CustomFragment", int64(2))
	val, ok := builder.Get("c8y_CustomFragment")
	testingutils.Equals(t, true, ok)
	testingutils.Equals(t, int64(2), val.(int64))

	_, ok = builder.Get("c8y_NonExistentProp")
	testingutils.Equals(t, false, ok)
}
