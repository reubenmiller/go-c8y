package api_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/devices/registration"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DeviceRequest_CreateGetDelete(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	deviceID := "TEST_DEVICE" + testingutils.RandomString(7)

	// Delete the request in case it already exists
	client.Devices.Registration.Delete(ctx, deviceID)

	// Create device request
	createResult := client.Devices.Registration.Create(ctx, registration.CreateOptions{
		ID: deviceID,
	})
	require.NoError(t, createResult.Err)
	assert.Equal(t, 201, createResult.HTTPStatus)
	assert.Equal(t, deviceID, createResult.Data.ID())

	// Get request
	getResult := client.Devices.Registration.Get(ctx, deviceID)
	require.NoError(t, getResult.Err)
	assert.Equal(t, 200, getResult.HTTPStatus)
	assert.Equal(t, deviceID, getResult.Data.ID())

	// Delete request
	deleteResult := client.Devices.Registration.Delete(ctx, deviceID)
	require.NoError(t, deleteResult.Err)
	assert.Equal(t, 204, deleteResult.HTTPStatus)
}

func Test_DeviceRequest_CRUD(t *testing.T) {
	// TODO: The fixed device bootstrap credentials are required to send the request for device credentials
	//       If no device credential requests are sent, then the device request remains in the AWAITING_CONNECTION state and can't be accepted
	t.Skip("The following requires device authentication")
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	deviceID := "TEST_DEVICE" + testingutils.RandomString(7)

	// Delete the request in case it already exists
	client.Devices.Registration.Delete(ctx, deviceID)

	// Create device request
	createResult := client.Devices.Registration.Create(ctx, registration.CreateOptions{
		ID: deviceID,
	})
	require.NoError(t, createResult.Err)
	assert.Equal(t, 201, createResult.HTTPStatus)
	assert.Equal(t, deviceID, createResult.Data.ID())

	// Update device request
	updateResult := client.Devices.Registration.Update(ctx, deviceID, registration.UpdateOptions{
		Status: string(types.DeviceRequestStatusPendingAccepted),
	})
	require.NoError(t, updateResult.Err)
	assert.Equal(t, 200, updateResult.HTTPStatus)
	assert.Equal(t, string(types.DeviceRequestStatusPendingAccepted), updateResult.Data.Status())

	// Delete request
	deleteResult := client.Devices.Registration.Delete(ctx, deviceID)
	require.NoError(t, deleteResult.Err)
	assert.Equal(t, 204, deleteResult.HTTPStatus)
}

func Test_PollNewDeviceRequest_TimeoutWithInvalidDeviceID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	done, err := client.Devices.Registration.PollNewDeviceRequest(
		ctx,
		"12345", // Invalid id
		1*time.Second,
		10*time.Second,
	)

	var doneCounter int64
	var errorCounter int64

	select {
	case <-done:
		atomic.AddInt64(&doneCounter, 1)

	case <-err:
		atomic.AddInt64(&errorCounter, 1)
	}

	assert.Equal(t, int64(1), errorCounter)
	assert.Equal(t, int64(0), doneCounter)
}
