package api_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/devices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects/childadditions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInventoryService_GetDevices(t *testing.T) {
	client := testcore.CreateTestClient(t)

	pageSize := 1
	result := client.Devices.List(context.Background(), devices.ListOptions{
		Query: "has(c8y_IsDevice)",
		PaginationOptions: pagination.PaginationOptions{
			PageSize: pageSize,
		},
	})

	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)

	devices, err := op.ToSliceR(result)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(devices), pageSize, "Should have at least %d device(s)", pageSize)

	if len(devices) > 0 {
		deviceName := devices[0].Name()
		assert.NotEmpty(t, deviceName)
		t.Logf("Device name: %s", deviceName)
	}
}

func TestInventoryService_AuthenticationToken(t *testing.T) {
	client := testcore.CreateTestClientNoAuth(t)

	pageSize := 1
	result := client.Devices.List(context.Background(), devices.ListOptions{
		PaginationOptions: pagination.PaginationOptions{
			PageSize: pageSize,
		},
	})

	assert.Error(t, result.Err)
	assert.Equal(t, http.StatusUnauthorized, result.HTTPStatus, "Expected unauthorized access response")
}

func TestInventoryService_CreateDevice(t *testing.T) {
	client := testcore.CreateTestClient(t)

	device := testcore.CreateDevice(t, client)
	assert.NoError(t, device.Err)
	assert.Equal(t, http.StatusCreated, device.HTTPStatus)
	assert.NotEmpty(t, device.Data.ID())
	assert.True(t, device.Data.Exists("c8y_IsDevice"), "Device should have c8y_IsDevice fragment")
}

func TestInventoryService_GetManagedObject(t *testing.T) {
	client := testcore.CreateTestClient(t)
	device := testcore.CreateDevice(t, client)

	retrieved := client.ManagedObjects.Get(context.Background(), device.Data.ID(), managedobjects.GetOptions{})
	assert.NoError(t, retrieved.Err)
	assert.Equal(t, http.StatusOK, retrieved.HTTPStatus)
	assert.Equal(t, device.Data.ID(), retrieved.Data.ID())
}

func TestInventoryService_UpdateManagedObject(t *testing.T) {
	client := testcore.CreateTestClient(t)
	device := testcore.CreateDevice(t, client)

	updatedName := "UpdatedDeviceName"
	update := map[string]any{
		"name": updatedName,
	}

	updated := client.ManagedObjects.Update(context.Background(), device.Data.ID(), update)
	assert.NoError(t, updated.Err)
	assert.Equal(t, http.StatusOK, updated.HTTPStatus)
	assert.Equal(t, updatedName, updated.Data.Name())
}

func TestInventoryService_DeleteManagedObject(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Create a device without auto-cleanup so we can test deletion explicitly
	device := client.Devices.Create(context.Background(), jsonmodels.NewDevice("ci"+testingutils.RandomString(16)))
	assert.NoError(t, device.Err)
	assert.Equal(t, http.StatusCreated, device.HTTPStatus)

	// Delete the device
	deleted := client.ManagedObjects.Delete(context.Background(), device.Data.ID(), managedobjects.DeleteOptions{})
	assert.NoError(t, deleted.Err)
	assert.Equal(t, http.StatusNoContent, deleted.HTTPStatus)

	// Verify it's gone
	retrieved := client.ManagedObjects.Get(context.Background(), device.Data.ID(), managedobjects.GetOptions{})
	assert.Error(t, retrieved.Err)
	assert.Equal(t, http.StatusNotFound, retrieved.HTTPStatus)
}

func TestInventoryService_GetChildAdditions(t *testing.T) {
	client := testcore.CreateTestClient(t)
	device := testcore.CreateDevice(t, client)
	client.Client.SetDebug(true)

	// Create child additions
	child01 := client.ManagedObjects.ChildAdditions.Create(
		context.Background(),
		device.Data.ID(),
		map[string]any{
			"name": "ntp",
			"type": "c8y_Service",
		},
	)
	assert.NoError(t, child01.Err)
	assert.Equal(t, http.StatusCreated, child01.HTTPStatus)

	child02 := client.ManagedObjects.ChildAdditions.Create(
		context.Background(),
		device.Data.ID(),
		map[string]any{
			"name": "mosquitto",
			"type": "c8y_Service",
		},
	)
	assert.NoError(t, child02.Err)
	assert.Equal(t, http.StatusCreated, child02.HTTPStatus)

	// Query for specific child
	items := client.ManagedObjects.ChildAdditions.List(
		context.Background(),
		device.Data.ID(),
		childadditions.ListOptions{
			Query: "$filter=(type eq 'c8y_Service' and name eq 'ntp')",
		},
	)
	assert.NoError(t, items.Err)
	assert.Equal(t, http.StatusOK, items.HTTPStatus)

	children, err := op.ToSliceR(items)
	assert.NoError(t, err)
	require.Len(t, children, 1)
	assert.Equal(t, child01.Data.ID(), children[0].ID())
}

// Binary operations have been migrated to port_binaries_test.go

func TestInventoryService_CreateManagedObjectWithBinary(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	testfile1 := testcore.NewDummyFile(t, "testfile1.txt", "test contents 1")

	// Create child addition and a binary
	binary1 := client.ManagedObjects.CreateWithBinary(context.Background(), managedobjects.CreateWithBinaryOptions{
		Body: map[string]any{
			"name": "MyConfigurationFile",
		},
		SetURLField:              true,
		URLFieldPath:             "url",
		AddChildAddition:         true,
		FailOnChildAdditionError: true,
		File: core.UploadFileOptions{
			FilePath: testfile1,
		},
	})
	assert.NoError(t, binary1.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), binary1.Data.ID(), managedobjects.DeleteOptions{
			ForceCascade: true,
		})
	})

	binaryURL := binary1.Data.Get("url").String()
	binaryID := binaryURL[strings.LastIndex(binaryURL, "/")+1:]

	binary := client.ManagedObjects.Get(context.Background(), binaryID, managedobjects.GetOptions{})
	assert.NoError(t, binary.Err)

	testingutils.Equals(t, "text/plain; charset=utf-8", binary.Data.Type())
	testingutils.Equals(t, "testfile1.txt", binary.Data.Name())

	testingutils.Equals(t, strings.ReplaceAll(binary.Data.Self(), "managedObjects", "binaries"), binary1.Data.Get("url").String())
}

func TestInventoryService_CreateChildAdditionWithBinary(t *testing.T) {
	client := testcore.CreateTestClient(t)
	parent := testcore.CreateManagedObject(t, client)
	client.Client.SetDebug(true)

	testfile1 := testcore.NewDummyFile(t, "testfile1.txt", "test contents 1")

	// Create parent

	// Create child addition and a binary
	child := client.ManagedObjects.CreateWithBinary(context.Background(), managedobjects.CreateWithBinaryOptions{
		Parent: parent.Data.ID(),
		Body: map[string]any{
			"name": "customChild",
		},
		SetURLField:              true,
		URLFieldPath:             "childUrl",
		AddChildAddition:         true,
		FailOnChildAdditionError: true,
		File: core.UploadFileOptions{
			FilePath: testfile1,
		},
	})
	assert.NoError(t, child.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), parent.Data.ID(), managedobjects.DeleteOptions{
			ForceCascade: true,
		})
	})

	childURL := child.Data.Get("childUrl").String()
	assert.NotEmpty(t, child.Data.ID(), "child id id should be set")
	assert.NotEmpty(t, childURL, "Child url should not be an empty string")

	binaryID := childURL[strings.LastIndex(childURL, "/")+1:]
	binary := client.ManagedObjects.Get(context.Background(), binaryID, managedobjects.GetOptions{})
	assert.NoError(t, binary.Err)

	testingutils.Equals(t, "text/plain; charset=utf-8", binary.Data.Type())
	testingutils.Equals(t, "testfile1.txt", binary.Data.Name())
	testingutils.Equals(t, strings.ReplaceAll(binary.Data.Self(), "managedObjects", "binaries"), childURL)
}
