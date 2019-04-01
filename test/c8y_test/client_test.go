package c8y_test

import (
	"context"
	"encoding/json"
	"log"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

// TestInventoryService_DecodeJSONManagedObject tests whether individual managed objects can be decoded into custom objects
func TestInventoryService_DecodeJSONManagedObject(t *testing.T) {
	client := createTestClient()

	pageSize := 1
	opt := &c8y.PaginationOptions{
		PageSize: pageSize,
	}

	data, _, _ := client.Inventory.GetDevices(context.Background(), opt)

	var mo c8y.ManagedObject

	err := json.Unmarshal([]byte(data.Items[0].Raw), &mo)

	log.Printf("Values: %s", mo)

	if err != nil {
		t.Errorf("Could not decode json. want: nil, got: %s", err)
	}
}

// TestInventoryService_DecodeJSONManagedObject tests whether the response from the server has be decoded to a custom object
func TestInventoryService_DecodeJSONManagedObjects(t *testing.T) {
	client := createTestClient()

	pageSize := 1
	opt := &c8y.PaginationOptions{
		PageSize: pageSize,
	}

	_, resp, _ := client.Inventory.GetDevices(context.Background(), opt)

	var apiResponse map[string]interface{}

	err := resp.DecodeJSON(&apiResponse)

	log.Printf("Values: %s", apiResponse)

	if err != nil {
		t.Errorf("Could not decode json. want: nil, got: %s", err)
	}
}
