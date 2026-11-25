package alarms_test

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/examplesetup"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
)

// Raise a major alarm on a device, list the major alarms and clear the alarm.
func Example() {
	client, stop := examplesetup.New()
	defer stop()
	ctx := context.Background()

	client.ManagedObjects.Create(ctx, map[string]any{"name": "sensor-01", "c8y_IsDevice": map[string]any{}})

	// The source can be an id, or a resolver reference such as a device name.
	created := client.Alarms.Create(ctx, alarms.CreateOptions{
		Source:   managedobjects.ByName("sensor-01"),
		Type:     "example_HighTemperature",
		Text:     "Temperature is too high",
		Severity: model.AlarmSeverityMajor,
	})
	if created.Err != nil {
		fmt.Println("create failed:", created.Err)
		return
	}
	fmt.Println("created:", created.Data.Text(), created.Data.Severity(), created.Data.Status())

	list := client.Alarms.List(ctx, alarms.ListOptions{
		Type:     []string{"example_HighTemperature"},
		Severity: []model.AlarmSeverity{model.AlarmSeverityMajor},
	})
	for alarm, err := range list.Items() {
		fmt.Println("major alarm:", alarm.Text(), err)
	}

	cleared := client.Alarms.Update(ctx, created.Data.ID(), map[string]any{"status": "CLEARED"})
	fmt.Println("updated:", cleared.Data.Status(), cleared.Err)
	// Output:
	// created: Temperature is too high MAJOR ACTIVE
	// major alarm: Temperature is too high <nil>
	// updated: CLEARED <nil>
}
