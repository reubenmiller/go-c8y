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
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
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
	collection := client.Measurements.List(
		context.Background(),
		measurements.ListOptions{
			DateFrom: time.Now().Add(-20 * 24 * time.Hour),
			DateTo:   time.Now(),
			PaginationOptions: pagination.PaginationOptions{
				PageSize:          1,
				WithTotalElements: true,
			},
		})

	// Always check for errors
	if collection.Err != nil {
		slog.Error("Could not retrieve alarms", "err", collection.Err)
		os.Exit(1)
	}

	slog.Info("Response", "status", collection.HTTPStatus, "duration", collection.Duration)

	// Generic iteration - access only common fields
	for measurement := range collection.Data.IterAs() {
		slog.Info("Measurement found", "id", measurement.ID(), "type", measurement.Type())
	}

	slog.Info("Measurements", "total", collection.Meta["totalElements"])

	//
	// Alarms
	alarmCollection := client.Alarms.List(context.Background(), alarms.ListOptions{
		DateFrom: time.Now().Add(-30 * 24 * time.Hour),
	})
	if alarmCollection.Err != nil {
		log.Panic(alarmCollection.Err)
	}
	slog.Info("Alarms", "total", alarmCollection.Data.Length())

	// Inventory binaries
	exampleCreateBinary(client)
}

func exampleCreateBinary(client *c8y_api.Client) error {
	binary := client.Binaries.Create(context.Background(), binaries.UploadFileOptions{
		Reader: strings.NewReader(`hello`),
		Name:   "unknown",
	})
	if binary.Err != nil {
		return binary.Err
	}

	slog.Info("Successfully created binary", "id", binary.Data.ID(), "lastUpdated(ago)", time.Since(binary.Data.LastUpdated()))

	if result := client.Binaries.Delete(context.TODO(), binary.Data.ID()); result.Err != nil {
		slog.Error("Failed to delete binary", "err", result.Err)
	}
	return nil
}
