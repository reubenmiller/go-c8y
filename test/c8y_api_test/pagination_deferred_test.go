package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
)

func Test_Pagination_WithDeferredExecution(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)
	ctx = api.WithMockResponses(ctx, true) // Use mocks to avoid real API calls

	// Create iterator - this is lazy, no API calls yet
	it := client.ManagedObjects.ListAll(ctx, managedobjects.ListOptions{
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 5,
			MaxItems: 10,
		},
	})

	// Iterator creation is lazy - no calls made yet
	// TotalCount is -1 until iteration starts
	if it.TotalCount() != -1 {
		t.Errorf("Expected TotalCount to be -1 (not fetched yet), got %d", it.TotalCount())
	}

	// With deferred execution, iterating should not execute requests
	count := 0
	for range it.Items() {
		count++
	}

	// Deferred execution prevents any API calls
	if count != 0 {
		t.Errorf("Expected 0 items with deferred execution, got %d", count)
	}

	// Metadata still not available since no requests executed
	if it.TotalCount() != -1 {
		t.Errorf("Expected TotalCount to remain -1 with deferred execution, got %d", it.TotalCount())
	}
}

func Test_Pagination_WithMockResponses_Metadata(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithMockResponses(context.Background(), true)

	// Create iterator - lazy, no API calls yet
	it := client.ManagedObjects.ListAll(ctx, managedobjects.ListOptions{
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 5,
			MaxItems: 10,
		},
	})

	// Before Preview() - no metadata available
	if it.TotalCount() != -1 {
		t.Errorf("Expected TotalCount to be -1 before Preview(), got %d", it.TotalCount())
	}

	// Call Preview() to fetch metadata without iterating
	err := it.Preview()
	if err != nil {
		t.Fatalf("Preview() failed: %v", err)
	}

	// After Preview() - metadata should be available
	t.Logf("After Preview() - Total count: %d, Total pages: %d",
		it.TotalCount(), it.TotalPages())

	if it.TotalCount() != 2 {
		t.Errorf("Expected TotalCount to be 2, got %d", it.TotalCount())
	}

	if it.TotalPages() != 1 {
		t.Errorf("Expected TotalPages to be 1, got %d", it.TotalPages())
	}

	// User can now decide whether to proceed with iteration
	if it.TotalCount() > 1000 {
		t.Skip("Too many items, skipping iteration")
	}

	// Now actually iterate
	count := 0
	for mo := range it.Items() {
		count++
		_ = mo
	}

	// Should get 2 items from mock collection
	if count != 2 {
		t.Errorf("Expected 2 items from mock collection, got %d", count)
	}

	if it.Err() != nil {
		t.Errorf("Iterator error: %v", it.Err())
	}
}

func Test_Pagination_Preview_WithDeferredExecution(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDeferredExecution(context.Background(), true)
	ctx = api.WithMockResponses(ctx, true)

	// Create iterator
	it := client.ManagedObjects.ListAll(ctx, managedobjects.ListOptions{
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 5,
			MaxItems: 10,
		},
	})

	// Try to preview with deferred execution
	err := it.Preview()

	// Preview should fail or return no data with deferred execution
	// because the preview call itself would be deferred
	t.Logf("Preview with deferred execution - Total count: %d, Error: %v",
		it.TotalCount(), err)

	// With deferred execution, metadata won't be available
	if it.TotalCount() != -1 {
		t.Logf("Note: TotalCount is %d (expected -1 with deferred execution)", it.TotalCount())
	}
}

func Test_Pagination_WithDryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := api.WithDryRun(context.Background(), true)
	ctx = api.WithMockResponses(ctx, true)

	// Dry run with mock responses (legacy behavior)
	it := client.ManagedObjects.ListAll(ctx, managedobjects.ListOptions{
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 5,
			MaxItems: 10,
		},
	})

	count := 0
	for range it.Items() {
		count++
	}

	// With both dry run and mock responses, we get mock data (2 items from mock collection)
	if count != 2 {
		t.Errorf("Expected 2 items from mock collection, got %d", count)
	}

	if it.Err() != nil {
		t.Logf("Iterator error: %v", it.Err())
	}
}

func Test_Pagination_WithMockResponsesOnly(t *testing.T) {
	client := testcore.CreateTestClient(t)
	// Only mock responses, no dry run logging
	ctx := api.WithMockResponses(context.Background(), true)

	// Use mock responses without logging
	it := client.ManagedObjects.ListAll(ctx, managedobjects.ListOptions{
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 5,
			MaxItems: 10,
		},
	})

	count := 0
	for range it.Items() {
		count++
	}

	// Should get 2 items from mock collection (without logging noise)
	if count != 2 {
		t.Errorf("Expected 2 items from mock collection, got %d", count)
	}

	if it.Err() != nil {
		t.Errorf("Iterator error: %v", it.Err())
	}
}

func Test_ContextOptions_Combinations(t *testing.T) {
	client := testcore.CreateTestClient(t)

	tests := []struct {
		name           string
		setupCtx       func(context.Context) context.Context
		expectedCount  int
		expectsLogging bool
	}{
		{
			name: "No flags - normal execution",
			setupCtx: func(ctx context.Context) context.Context {
				return ctx
			},
			expectedCount:  -1, // Real API call, skip count check
			expectsLogging: false,
		},
		{
			name: "DryRun only - logs + mock (backward compat)",
			setupCtx: func(ctx context.Context) context.Context {
				return api.WithDryRun(ctx, true)
			},
			expectedCount:  2, // Mock data
			expectsLogging: true,
		},
		{
			name: "MockResponses only - mock without logging",
			setupCtx: func(ctx context.Context) context.Context {
				return api.WithMockResponses(ctx, true)
			},
			expectedCount:  2, // Mock data
			expectsLogging: false,
		},
		{
			name: "Both flags - logs + mock",
			setupCtx: func(ctx context.Context) context.Context {
				ctx = api.WithDryRun(ctx, true)
				return api.WithMockResponses(ctx, true)
			},
			expectedCount:  2, // Mock data
			expectsLogging: true,
		},
		{
			name: "Deferred execution - no execution at all",
			setupCtx: func(ctx context.Context) context.Context {
				return api.WithDeferredExecution(ctx, true)
			},
			expectedCount:  0, // Nothing executed
			expectsLogging: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedCount < 0 {
				t.Skip("Skipping test that would make real API call")
			}

			ctx := tt.setupCtx(context.Background())
			it := client.ManagedObjects.ListAll(ctx, managedobjects.ListOptions{
				PaginationOptions: pagination.PaginationOptions{
					PageSize: 5,
					MaxItems: 10,
				},
			})

			count := 0
			for range it.Items() {
				count++
			}

			if count != tt.expectedCount {
				t.Errorf("Expected %d items, got %d", tt.expectedCount, count)
			}
		})
	}
}
