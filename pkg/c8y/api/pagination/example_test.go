package pagination_test

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/examplesetup"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
)

// ListAll returns an iterator which fetches pages on demand. PageSize sets the
// size of each request and MaxItems stops the iteration after that many items.
func ExampleIterator() {
	client, stop := examplesetup.New()
	defer stop()
	ctx := context.Background()

	for i := 1; i <= 7; i++ {
		client.ManagedObjects.Create(ctx, map[string]any{"name": fmt.Sprintf("sensor-%02d", i), "type": "example_Sensor"})
	}

	// Walk every event, two per request.
	count := 0
	opts := managedobjects.ListOptions{Type: "example_Sensor", PaginationOptions: pagination.PaginationOptions{PageSize: 2}}
	for _, err := range client.ManagedObjects.ListAll(ctx, opts).Items() {
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		count++
	}
	fmt.Println("all:", count)

	// Stop after three items.
	opts.MaxItems = 3
	count = 0
	for range client.ManagedObjects.ListAll(ctx, opts).Items() {
		count++
	}
	fmt.Println("limited:", count)
	// Output:
	// all: 7
	// limited: 3
}
