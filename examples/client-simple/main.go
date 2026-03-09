package main

import (
	"context"
	"log"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

type CustomField struct {
	URL string `json:"url,omitempty"`
}

type ExampleMO struct {
	c8y.ManagedObject

	CustomField *CustomField `json:"my_CustomField,omitempty"`
}

func main() {
	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	client := c8y.NewClientFromEnvironment(nil, false)

	// Get list of alarms with MAJOR Severity
	alarmCollection, _, err := client.Alarm.GetAlarms(
		context.Background(),
		&c8y.AlarmCollectionOptions{
			Severity: "MAJOR",
		},
	)

	// Always check for errors
	if err != nil {
		log.Fatalf("Could not retrieve alarms. %s", err)
	}

	log.Printf("Total alarms: %d", len(alarmCollection.Alarms))

	// Create custom managed object
	mo := &ExampleMO{
		ManagedObject: c8y.ManagedObject{
			Name: "hello",
		},
		CustomField: &CustomField{
			URL: "example.com/foo",
		},
	}
	_, resp, err := client.Inventory.Create(context.Background(), mo)
	if err != nil {
		log.Fatalf("Failed to create the managed object. %s", err)
	}

	log.Printf("mo: %v", resp.JSON().Raw)
}
