package operations_test

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/examplesetup"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/operations"
)

// Create a restart operation for a device, then mark it as successful as the device would.
func Example() {
	client, stop := examplesetup.New()
	defer stop()
	ctx := context.Background()

	client.ManagedObjects.Create(ctx, map[string]any{"name": "sensor-01", "c8y_IsDevice": map[string]any{}})

	created := client.Operations.Create(ctx, operations.CreateOptions{
		DeviceID:    managedobjects.ByName("sensor-01"),
		Description: "Restart device",
		Fragments:   []model.Fragment{model.Frag("c8y_Restart", map[string]any{})},
	})
	if created.Err != nil {
		fmt.Println("create failed:", created.Err)
		return
	}
	fmt.Println("created:", created.Data.Status(), created.Data.Description())

	done := client.Operations.Update(ctx, created.Data.ID(), map[string]any{"status": "SUCCESSFUL"})
	fmt.Println("updated:", done.Data.Status(), done.Err)
	// Output:
	// created: PENDING Restart device
	// updated: SUCCESSFUL <nil>
}
