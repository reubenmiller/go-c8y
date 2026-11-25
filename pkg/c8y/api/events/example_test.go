package events_test

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/examplesetup"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/events"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
)

// Create an event with a custom fragment and list the events of a type.
func Example() {
	client, stop := examplesetup.New()
	defer stop()
	ctx := context.Background()

	device := client.ManagedObjects.Create(ctx, map[string]any{"name": "sensor-01", "c8y_IsDevice": map[string]any{}})

	created := client.Events.Create(ctx, events.CreateOptions{
		Source:    managedobjects.ByID(device.Data.ID()),
		Type:      "example_DoorOpened",
		Text:      "Door opened",
		Fragments: []model.Fragment{model.Frag("example_Door", map[string]any{"id": 7})},
	})
	if created.Err != nil {
		fmt.Println("create failed:", created.Err)
		return
	}
	fmt.Println("created:", created.Data.Text(), created.Data.Get("example_Door.id").Int())

	for event, err := range client.Events.ListAll(ctx, events.ListOptions{Type: "example_DoorOpened"}).Items() {
		fmt.Println("listed:", event.Text(), err)
	}
	// Output:
	// created: Door opened 7
	// listed: Door opened <nil>
}
