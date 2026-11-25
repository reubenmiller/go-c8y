package measurements_test

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/examplesetup"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/measurements"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
)

// Create a temperature measurement and list the measurements of a device.
func Example() {
	client, stop := examplesetup.New()
	defer stop()
	ctx := context.Background()

	device := client.ManagedObjects.Create(ctx, map[string]any{"name": "sensor-01", "c8y_IsDevice": map[string]any{}})
	source := managedobjects.ByID(device.Data.ID())

	created := client.Measurements.Create(ctx, measurements.CreateOptions{
		Source: source,
		Type:   "c8y_TemperatureMeasurement",
		Fragments: []model.Fragment{
			model.Frag("c8y_Temperature", map[string]any{"T": map[string]any{"value": 21.5, "unit": "C"}}),
		},
	})
	if created.Err != nil {
		fmt.Println("create failed:", created.Err)
		return
	}

	for m, err := range client.Measurements.ListAll(ctx, measurements.ListOptions{Source: source}).Items() {
		fmt.Println(m.Type(), m.Get("c8y_Temperature.T.value").Float(), err)
	}
	// Output:
	// c8y_TemperatureMeasurement 21.5 <nil>
}
