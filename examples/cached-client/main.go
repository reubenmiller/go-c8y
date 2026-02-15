package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
)

func main() {
	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	cacheDir := filepath.Join(os.TempDir(), "go-c8y-cache")
	// rt := http.DefaultTransport
	cached := c8y.NewCachedTransport(nil, 5*time.Second, cacheDir, nil, c8y.CacheOptions{})

	client := api.NewClient(api.ClientOptions{
		BaseURL:   authentication.HostFromEnvironment(),
		Auth:      authentication.FromEnvironment(),
		Transport: cached,
	})

	// Get list of alarms with MAJOR Severity
	alarmResult := client.Alarms.List(
		context.Background(),
		alarms.ListOptions{
			Severity: []model.AlarmSeverity{
				model.AlarmSeverityMajor,
			},
		},
	)

	// Always check for errors
	if alarmResult.Err != nil {
		slog.Error("Could not retrieve alarms", "err", alarmResult.Err)
		os.Exit(1)
	}

	// Check if response was from cache using Meta map
	cacheStatus := alarmResult.Meta["x-cache"]
	fromCache := alarmResult.Meta["x-from-cache"]
	age := alarmResult.Meta["age"]

	slog.Info("First request",
		"total", alarmResult.Data.Length(),
		"cache", cacheStatus,
		"from_cache", fromCache,
		"age_seconds", age,
	)

	alarmResult2 := client.Alarms.List(
		context.Background(),
		alarms.ListOptions{
			Severity: []model.AlarmSeverity{
				model.AlarmSeverityMajor,
			},
		},
	)

	if alarmResult2.Err != nil {
		slog.Error("Could not retrieve alarms", "err", alarmResult2.Err)
		os.Exit(1)
	}

	// Check if response was from cache using Meta map
	cacheStatus2 := alarmResult2.Meta["x-cache"]
	fromCache2 := alarmResult2.Meta["x-from-cache"]
	age2 := alarmResult2.Meta["age"]

	slog.Info("Second request (should be cached)",
		"total", alarmResult2.Data.Length(),
		"cache", cacheStatus2,
		"from_cache", fromCache2,
		"age_seconds", age2,
	)
}
