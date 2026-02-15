package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/operations"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
)

func Test_Operations_DeviceResolver_ByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	opts := operations.ListOptions{
		DeviceID: client.Operations.DeviceResolver.ByName("device01"),
	}

	result := client.Operations.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed operations with device name resolver")
}

func Test_Operations_DeviceResolver_AgentID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	opts := operations.ListOptions{
		AgentID: client.Operations.DeviceResolver.ByExternalID("c8y_Serial", "ABC123"),
	}

	result := client.Operations.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed operations with agent ID resolver")
}

func Test_Operations_DeviceResolver_StringBased(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	opts := operations.ListOptions{
		DeviceID: "name:device01",
	}

	result := client.Operations.List(ctx, opts)
	if result.Err != nil {
		t.Fatalf("List failed: %v", result.Err)
	}

	t.Logf("Successfully listed operations with string-based device resolver")
}
