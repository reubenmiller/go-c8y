package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
)

func Test_Microservices_ByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	// Test ByName returns a string reference
	ref := client.Microservices.ByName("my-microservice")
	if ref != "name:my-microservice" {
		t.Errorf("Expected ref to be 'name:my-microservice', got '%s'", ref)
	}

	// Test ResolveID with the name reference
	meta := make(map[string]any)
	id, err := client.Microservices.ResolveID(ctx, ref, meta)
	if err != nil {
		t.Fatalf("ResolveID failed: %v", err)
	}

	if id == "" {
		t.Error("Expected resolved ID to be non-empty")
	}

	if meta["name"] == "" {
		t.Error("Expected metadata to contain name")
	}

	t.Logf("Resolved ID: %s, Metadata: %+v", id, meta)
}

func Test_Microservices_ByContextPath(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	// Test ByContextPath returns a string reference
	ref := client.Microservices.ByContextPath("/my-microservice")
	if ref != "contextPath:/my-microservice" {
		t.Errorf("Expected ref to be 'contextPath:/my-microservice', got '%s'", ref)
	}

	// Test ResolveID with the contextPath reference
	meta := make(map[string]any)
	id, err := client.Microservices.ResolveID(ctx, ref, meta)
	if err != nil {
		t.Fatalf("ResolveID failed: %v", err)
	}

	if id == "" {
		t.Error("Expected resolved ID to be non-empty")
	}

	if meta["contextPath"] == "" {
		t.Error("Expected metadata to contain contextPath")
	}

	t.Logf("Resolved ID: %s, Metadata: %+v", id, meta)
}

func Test_Microservices_ByID(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Test ByID returns the ID directly
	ref := client.Microservices.ByID("12345")
	if ref != "12345" {
		t.Errorf("Expected ref to be '12345', got '%s'", ref)
	}
}

func Test_Microservices_ResolveID_NotFound(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	// Test with non-existent name
	ref := client.Microservices.ByName("nonexistent-microservice")
	_, err := client.Microservices.ResolveID(ctx, ref, nil)
	if err == nil {
		t.Error("Expected error for non-existent microservice")
	}

	t.Logf("Expected error: %v", err)
}

func Test_Microservices_Get_WithResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	// Test Get with name resolver
	result := client.Microservices.Get(ctx, client.Microservices.ByName("my-microservice"))
	if result.Err != nil {
		t.Fatalf("Get failed: %v", result.Err)
	}

	microservice := result.Data
	if microservice.ID() != "44444" {
		t.Errorf("Expected microservice ID to be '44444', got '%s'", microservice.ID())
	}

	if microservice.Name() != "my-microservice" {
		t.Errorf("Expected microservice name to be 'my-microservice', got '%s'", microservice.Name())
	}

	t.Logf("Successfully retrieved microservice: %s (ID: %s)", microservice.Name(), microservice.ID())
}

func Test_Microservices_Get_WithContextPathResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	// Test Get with contextPath resolver
	result := client.Microservices.Get(ctx, client.Microservices.ByContextPath("/my-microservice"))
	if result.Err != nil {
		t.Fatalf("Get failed: %v", result.Err)
	}

	microservice := result.Data
	if microservice.ContextPath() != "/my-microservice" {
		t.Errorf("Expected contextPath to be '/my-microservice', got '%s'", microservice.ContextPath())
	}

	t.Logf("Successfully retrieved microservice by contextPath: %s", microservice.ContextPath())
}
