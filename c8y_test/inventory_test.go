package c8y_test

import (
	"context"
	"fmt"
	"log"
	"testing"

	c8y "github.com/reubenmiller/go-c8y"
)

func getDevices(client *c8y.Client, name string, pageSize int) (*c8y.ManagedObjectCollection, *c8y.Response, error) {

	opt := &c8y.ManagedObjectOptions{
		Query: fmt.Sprintf("has(c8y_IsDevice) and (name eq '%s')", name),
		PaginationOptions: c8y.PaginationOptions{
			PageSize: pageSize,
		},
	}
	col, resp, err := client.Inventory.GetManagedObjectCollection(context.Background(), opt)

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
