package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alarms"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/binaries"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/measurements"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/tidwall/gjson"
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
	client := c8y_api.NewClient(c8y_api.ClientOptions{
		BaseURL: authentication.HostFromEnvironment(),
		Auth:    authentication.FromEnvironment(),
	})

	if enabledDebug, err := strconv.ParseBool(os.Getenv("DEBUG")); err == nil {
		client.Client.SetDebug(enabledDebug)
	}

	// Get list of measurements
	collection := new(model.MeasurementCollection)
	resp, err := client.Measurements.ListB(measurements.ListOptions{
		DateFrom: time.Now().Add(-20 * 24 * time.Hour),
		DateTo:   time.Now(),
		PaginationOptions: pagination.PaginationOptions{
			PageSize:          1,
			WithTotalElements: true,
		},
	}).
		SetResponseBodyUnlimitedReads(true).
		SetResult(collection).
		SetContext(context.Background()).
		Send()

	// Always check for errors
	if err != nil {
		slog.Error("Could not retrieve alarms", "err", err)
		os.Exit(1)
	}
	slog.Info("Response", "status", resp.Response.Status(), "duration", resp.Response.Duration())
	// Or access the raw json (in addition to the setting the result)
	raw := gjson.Parse(resp.String())
	slog.Info("Measurement found", "id", raw.Get("measurements.0.id").String(), "type", raw.Get("measurements.0.type").String())

	slog.Info("Measurements", "total", collection.Statistics.TotalElements)

	//
	// Alarms
	alarmCollection, err := client.Alarms.List(context.Background(), alarms.ListOptions{
		DateFrom: time.Now().Add(-30 * 24 * time.Hour),
	})
	if err != nil {
		log.Panic(err)
	}
	slog.Info("Alarms", "total", len(alarmCollection.Alarms))

	// Inventory binaries
	exampleCreateBinary(client)
}

func exampleCreateBinary(client *c8y_api.Client) error {
	binary, err := client.Binaries.Create(context.Background(), binaries.UploadFileOptions{
		Reader: strings.NewReader(`hello`),
		Name:   "unknown",
	})
	if err != nil {
		return err
	}

	slog.Info("Successfully created binary", "id", binary.ID, "lastUpdated(ago)", time.Since(binary.LastUpdated))

	if err := client.Binaries.Delete(context.TODO(), binary.ID); err != nil {
		slog.Error("Failed to delete binary", "err", err)
	}
	return nil
}
