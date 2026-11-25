package managedobjects_test

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/examplesetup"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
)

// Create a managed object, read it back, list by type and resolve a name to an id.
func Example() {
	client, stop := examplesetup.New()
	defer stop()
	ctx := context.Background()

	created := client.ManagedObjects.Create(ctx, map[string]any{
		"name": "sensor-01", "type": "example_Sensor", "c8y_IsDevice": map[string]any{},
	})
	if created.Err != nil {
		fmt.Println("create failed:", created.Err)
		return
	}
	fmt.Println("created:", created.Data.Get("name").String(), created.HTTPStatus)

	id := created.Data.Get("id").String()
	got := client.ManagedObjects.Get(ctx, id, managedobjects.GetOptions{})
	fmt.Println("got:", got.Data.Get("name").String())

	// ListAll walks every page; the iterator reports errors as the second value.
	for mo, err := range client.ManagedObjects.ListAll(ctx, managedobjects.ListOptions{Type: "example_Sensor"}).Items() {
		if err != nil {
			fmt.Println("list failed:", err)
			return
		}
		fmt.Println("listed:", mo.Get("name").String())
	}

	// A "name:" prefix selects the lookup. A plain value is taken as an id.
	resolved, err := client.ManagedObjects.ResolveID(ctx, "name:sensor-01", nil)
	fmt.Println("resolved to the same id:", resolved == id, err)
	// Output:
	// created: sensor-01 201
	// got: sensor-01
	// listed: sensor-01
	// resolved to the same id: true <nil>
}
