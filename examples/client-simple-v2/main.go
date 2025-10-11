package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/binaries"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/measurements"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
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
	client := c8y_api.NewClientV2(c8y_api.ClientOptions{
		BaseURL:  "https://" + os.Getenv("C8Y_HOST"),
		Username: os.Getenv("C8Y_USERNAME"),
		Password: os.Getenv("C8Y_PASSWORD"),
	})

	if enabledDebug, err := strconv.ParseBool(os.Getenv("DEBUG")); err == nil {
		client.Client.SetDebug(enabledDebug)
	}

	// Get list of measurements
	collection := new(measurements.MeasurementCollection)
	resp, err := client.Measurement.List(context.Background(), &measurements.ListOptions{
		DateFrom: time.Now().Add(-20 * 24 * time.Hour),
		DateTo:   time.Now(),
		PaginationOptions: pagination.PaginationOptions{
			PageSize:          1,
			WithTotalElements: true,
		},
	}).
		SetResponseBodyUnlimitedReads(true).
		SetResult(collection).
		Send()

	// Always check for errors
	if err != nil {
		slog.Error("Could not retrieve alarms", "err", err)
		os.Exit(1)
	}

	slog.Info("Response", "status", resp.Status(), "duration", resp.Duration())

	slog.Info("Measurements", "total", collection.Statistics.TotalElements)

	// Or access the raw json (in addition to the setting the result)
	raw := gjson.Parse(resp.String())
	slog.Info("Measurement found", "id", raw.Get("measurements.0.id").String(), "type", raw.Get("measurements.0.type").String())

	exampleCreateBinary(client)

	exampleChildAdditions(client)
}

func exampleCreateBinary(client *c8y_api.Client) error {
	binary := &binaries.BinaryManagedObject{}
	_, err := client.InventoryBinary.Create(context.Background(), binaries.CreateOptions{
		File: &resty.MultipartField{
			Reader: strings.NewReader(`hello`),
		},
	}).SetResult(binary).Send()
	if err != nil {
		return err
	}

	slog.Info("Successfully created binary", "id", binary.ID, "lastUpdated(ago)", time.Since(binary.LastUpdated))

	if resp, err := client.InventoryBinary.Delete(context.TODO(), binary.ID).Send(); err != nil {
		slog.Error("Failed to delete binary", "err", err, "response", resp.String())
	}
	return nil
}

func exampleChildAdditions(client *c8y_api.Client) error {
	_ = client.ManagedObjects.ChildAdditions.Assign(context.Background(), "1234", nil).Funcs(c8y_api.SetProcessingModeCEP())
	return nil
}
