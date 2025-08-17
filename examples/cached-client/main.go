package main

import (
	"context"
	"log/slog"
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
		slog.Error("Could not retrieve alarms", "err", err)
		panic(err)
	}

	slog.Info("Alarms", "total", len(alarmCollection.Alarms))

	alarmCollection2, _, err2 := client.Alarm.GetAlarms(
		context.Background(),
		&c8y.AlarmCollectionOptions{
			Severity: "MAJOR",
		},
	)

	if err2 != nil {
		slog.Error("Could not retrieve alarms", "err", err2)
		panic(err2)
	}

	slog.Info("Alarms", "total", len(alarmCollection2.Alarms))
}
