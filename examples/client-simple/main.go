package main

import (
	"context"
	"log"

	c8y "github.com/reubenmiller/go-c8y"
)

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
}
