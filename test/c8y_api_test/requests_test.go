package c8y_api_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_ErrorHandlingGet(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)
	mo := client.ManagedObjects.Get(context.Background(), "0", managedobjects.GetOptions{})
	assert.Error(t, mo.Err)
	assert.Equal(t, 0, mo.Data.Length())
}

func Test_ErrorHandlingCreateEvent(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	evt := client.Events.Create(context.Background(), model.Event{
		Source: model.NewSource("0"),
	})
	err := evt.Err
	assert.Error(t, err)
	assert.Equal(t, 0, evt.Data.Length())

	assert.True(t, c8y_api.ErrHasStatus(err, 422))
	assert.True(t, errors.Is(err, c8y_api.Error{Code: 422}))
	assert.True(t, errors.Is(err, c8y_api.ErrUnprocessableEntity))
	sdkError := err.(*c8y_api.Error)
	assert.NotEmpty(t, sdkError.MessageRaw)

}

func Test_SimpleDefaultRequest(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	body := map[string]any{
		"name": "custom",
	}

	mo := client.ManagedObjects.Create(context.Background(), body)
	assert.NoError(t, mo.Err)
	assert.Greater(t, mo.Data.Length(), 0)

	assert.NotEmpty(t, mo.Data.ID())
	assert.Equal(t, mo.Data.Name(), "custom")
}

func Test_CreateRequestWithCustomHeaders(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	body := map[string]any{
		"name": "custom",
	}

	out := map[string]any{}
	resp, err := client.ManagedObjects.CreateB(body).
		SetProcessingMode("QUIESCENT").
		SetContext(context.Background()).
		SetResult(&out).
		Send()

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Contains(t, out, "id")
	assert.Equal(t, out["name"].(string), "custom")
}

func Test_CreateRequestDecodeHelper(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	body := map[string]any{
		"name": "custom",
	}

	resp, err := client.ManagedObjects.CreateB(body).
		SetContext(context.Background()).
		Send()

	assert.NoError(t, err)
	assert.NotNil(t, resp)

	out := make(map[string]any)
	err = json.Unmarshal([]byte(resp.String()), &out)

	// err = c8y_api.UnmarshalJSON(resp, &out)
	assert.NoError(t, err)
	assert.Contains(t, out, "id")
	assert.Equal(t, out["name"].(string), "custom")
}

func Test_CreateResultWithCustomResult(t *testing.T) {

	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	type bodyT struct {
		model.ManagedObject
		Model string `json:"model,omitempty"`
		Name  string `json:"display,omitempty"`
	}
	body := &bodyT{
		ManagedObject: model.ManagedObject{
			Name: "custom",
		},
		Name:  "Custom Device",
		Model: "linuxA",
	}

	result, resp, err := c8y_api.Execute[bodyT](context.Background(), client.ManagedObjects.CreateB(body))
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, result.ID)
	assert.NotEmpty(t, resp.String())
	assert.Equal(t, result.ManagedObject.Name, "custom")
	assert.Equal(t, result.Name, "Custom Device")
	assert.Equal(t, result.Model, "linuxA")
}
