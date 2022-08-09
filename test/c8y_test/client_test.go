package c8y_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
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

	if err != nil {
		t.Errorf("Could not decode json. want: nil, got: %s", err)
	}
}

func Test_SendRequest(t *testing.T) {
	options := c8y.RequestOptions{
		Host:   "https://c8y.example/base/",
		Method: "GET",
		Path:   "/path/with space?query=test eq%20'me'",
		Query:  "pageSize=100&another=%20again ",
	}

	currentPath, err := options.GetEscapedPath()
	if err != nil {
		t.Errorf("Invalid path. want: nil, got: %s", err)
	}

	if currentPath != "/base/path/with%20space" {
		t.Errorf("Path does not match. %s", currentPath)
	}

	currentQuery, err := options.GetQuery()
	if err != nil {
		t.Errorf("Invalid query. want: nil, got: %s", err)
	}

	if currentQuery != "another=+again+&pageSize=100&query=test+eq+%27me%27" {
		t.Errorf("Query does not match. %s", currentQuery)
	}
}

func Test_SendRequest_Get(t *testing.T) {
	client := createTestClient()

	options := c8y.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
	}
	resp, err := client.SendRequest(context.Background(), options)

	testingutils.Ok(t, err)

	if len(resp.Body()) == 0 {
		t.Errorf("received empty body. got=0, wanted=!0")
	}
}
