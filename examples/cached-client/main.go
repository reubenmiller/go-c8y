package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func main() {
	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	cacheDir := filepath.Join(os.TempDir(), "go-c8y-cache")
	httpClient := c8y.NewCachedClient(c8y.NewHTTPClient(
		c8y.WithInsecureSkipVerify(false),
	), cacheDir, 5*time.Second, nil, c8y.CacheOptions{})
	client := c8y.NewClientFromEnvironment(httpClient, false)

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

	alarmCollection2, _, err2 := client.Alarm.GetAlarms(
		context.Background(),
		&c8y.AlarmCollectionOptions{
			Severity: "MAJOR",
		},
	)

	if err2 != nil {
		log.Fatalf("Could not retrieve alarms. %s", err)
	}

	log.Printf("Total alarms: %d", len(alarmCollection2.Alarms))
}
