package c8y_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
)

func TestDeviceCredentialsService_PollNewDeviceRequest_CreateGetDelete(t *testing.T) {
	client := createTestClient()

	deviceID := "TEST_DEVICE" + testingutils.RandomString(7)

	// Delete the request in case it already exists
	client.DeviceCredentials.Delete(
		context.Background(),
		deviceID,
	)

	//
	// Create device request
	deviceReq, resp, err := client.DeviceCredentials.Create(
		context.Background(),
		deviceID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Equals(t, deviceID, deviceReq.ID)

	//
	// Get request
	getDeviceReq, resp, err := client.DeviceCredentials.GetNewDeviceRequest(
		context.Background(),
		deviceID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, deviceID, getDeviceReq.ID)

	//
	// Delete request
	resp, err = client.DeviceCredentials.Delete(
		context.Background(),
		deviceID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())
}

func TestDeviceCredentialsService_PollNewDeviceRequest_CRUD(t *testing.T) {
	// TODO: The fixed device bootstrap credentials are required to send the request for device credentials
	//       If no device credential requests are sent, then the device request remains in the AWAITING_CONNECTION state and can't be accepted
	t.Skip("The following requires device authentication")
	client := createTestClient()

	deviceID := "TEST_DEVICE" + testingutils.RandomString(7)

	// Delete the request in case it already exists
	client.DeviceCredentials.Delete(
		context.Background(),
		deviceID,
	)

	//
	// Create device request
	deviceReq, resp, err := client.DeviceCredentials.Create(
		context.Background(),
		deviceID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Equals(t, deviceID, deviceReq.ID)

	//
	// Update device request
	updatedDeviceReq, resp, err := client.DeviceCredentials.Update(
		context.Background(),
		deviceID,
		c8y.NewDeviceRequestAccepted,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, c8y.NewDeviceRequestAccepted, updatedDeviceReq.Status)

	//
	// Delete request
	resp, err = client.DeviceCredentials.Delete(
		context.Background(),
		deviceID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())
}

func TestDeviceCredentialsService_PollNewDeviceRequest_TimeoutWithInvalidDeviceID(t *testing.T) {
	client := createTestClient()

	done, err := client.DeviceCredentials.PollNewDeviceRequest(
		context.Background(),
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

	testingutils.Equals(t, int64(1), errorCounter)
	testingutils.Equals(t, int64(0), doneCounter)
}
