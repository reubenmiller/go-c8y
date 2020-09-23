package c8y_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func getDevices(client *c8y.Client, name string, pageSize int) (*c8y.ManagedObjectCollection, *c8y.Response, error) {

	opt := &c8y.ManagedObjectOptions{
		Query: fmt.Sprintf("has(c8y_IsDevice) and (name eq '%s')", name),
		PaginationOptions: c8y.PaginationOptions{
			PageSize: pageSize,
		},
	}
	col, resp, err := client.Inventory.GetManagedObjects(context.Background(), opt)

	return col, resp, err
}

func TestInventoryService_GetDevices(t *testing.T) {
	client := createTestClient()

	pageSize := 1
	opt := &c8y.PaginationOptions{
		PageSize: pageSize,
	}
	data, _, _ := client.Inventory.GetDevices(context.Background(), opt)

	if len(data.Items) != pageSize {
		t.Errorf("Unexpected amount of devices found. want %d, got: %d", pageSize, len(data.Items))
	}

	deviceName := data.Items[0].Get("name")

	log.Printf("Device Name: %s\n", deviceName)
}

func TestInventoryService_AuthenticationToken(t *testing.T) {
	client := createTestClient()

	pageSize := 1
	opt := &c8y.PaginationOptions{
		PageSize: pageSize,
	}
	// Throw invalid credentials
	ctx := c8y.NewAuthorizationContext("test", "something", "value")
	_, resp, err := client.Inventory.GetDevices(ctx, opt)

	if resp.StatusCode != 401 {
		t.Errorf("Expected unauthorized access response. want: 401, got: %d", resp.StatusCode)
	}

	if err == nil {
		t.Errorf("Function should have thrown an error. %s", err)
	}
}

func TestInventoryService_CreateUpdateDeleteBinary(t *testing.T) {
	client := createTestClient()

	testfile1 := NewDummyFile("testfile1", "test contents 1")
	testfile2 := NewDummyFile("testfile2", "test contents 2")

	// Configure required properties
	fileProperties := map[string]string{
		"name": "filename",
		"type": "text/plain",
	}

	defer func() {
		os.Remove(testfile1)
		os.Remove(testfile2)
	}()

	// Upload a new binary
	binary1, resp, err := client.Inventory.CreateBinary(context.Background(), testfile1, fileProperties)
	testingutils.Ok(t, err)
	testingutils.Assert(t, binary1.ID != "", "Binary ID should not be an empty string")
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode)

	// Download the binary, and check if it matches the file that was uploaded exactly
	downloadedBinary1, err := client.Inventory.DownloadBinary(context.Background(), binary1.ID)
	testingutils.Ok(t, err)
	defer os.Remove(downloadedBinary1)
	testingutils.FileEquals(t, testfile1, downloadedBinary1)

	// Update the existing binary with a new binary
	binary2, resp, err := client.Inventory.UpdateBinary(context.Background(), binary1.ID, testfile2)
	testingutils.Ok(t, err)

	// testingutils.Assert(t, binary1.ID != binary2.ID, "Binary ID should change if the binary has been updated")
	testingutils.Assert(t, binary2.ID != "", "Binary id should not be an empty string")

	// Download the updated binary and check if it matches the new binary contents
	downloadedBinary2, err := client.Inventory.DownloadBinary(context.Background(), binary2.ID)
	testingutils.Ok(t, err)

	defer os.Remove(downloadedBinary2)
	testingutils.FileEquals(t, testfile2, downloadedBinary2)

	// Delete the binary
	resp, err = client.Inventory.DeleteBinary(context.Background(), binary2.ID)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode)

	// Check if the managed object was deleted
	_, resp, err = client.Inventory.GetManagedObject(context.Background(), binary2.ID, nil)
	testingutils.Equals(t, http.StatusNotFound, resp.StatusCode)
	testingutils.Assert(t, err != nil, "Error should contain additional information about the request")

	// Check if the binary was deleted
	downloadedBinary3, err := client.Inventory.DownloadBinary(context.Background(), binary2.ID)
	testingutils.Assert(t, err != nil, "Error should contain additional information about the request")
	testingutils.Equals(t, "", downloadedBinary3)
}
