package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/destel/rill"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/binaries"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/measurements"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

type CustomField struct {
	URL string `json:"url,omitempty"`
}

type ExampleMO struct {
	model.ManagedObject

	CustomField *CustomField `json:"my_CustomField,omitempty"`
}

func main() {
	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	client := api.NewClient(api.ClientOptions{
		BaseURL: authentication.HostFromEnvironment(),
		Auth:    authentication.FromEnvironment(),
	})

	if enabledDebug, err := strconv.ParseBool(os.Getenv("DEBUG")); err == nil {
		client.SetDebug(enabledDebug)
	}

	// Get list of measurements (without pagination)
	measurementsResult := client.Measurements.List(
		context.Background(),
		measurements.ListOptions{
			DateFrom: time.Now().Add(-20 * 24 * time.Hour),
			DateTo:   time.Now(),
			Source:   managedobjects.ByExternalID("c8y_Serial", "rpi4-d83add90fe56"), // resolve by external id
			PaginationOptions: pagination.PaginationOptions{
				PageSize:          1,
				WithTotalElements: true,
			},
		})

	// Always check for errors
	if measurementsResult.Err != nil {
		slog.Error("Could not retrieve alarms", "err", measurementsResult.Err)
		os.Exit(1)
	}

	slog.Info("Response", "status", measurementsResult.HTTPStatus, "duration", measurementsResult.Duration)

	// Generic iteration - access only common fields
	for measurement := range op.Iter(measurementsResult) {
		slog.Info("Measurement found", "id", measurement.ID(), "type", measurement.Type())
	}

	slog.Info("Measurements", "total", measurementsResult.TotalElements())

	//
	// Alarms
	alarmCollection := client.Alarms.ListAll(context.Background(), alarms.ListOptions{
		DateFrom: time.Now().Add(-30 * 24 * time.Hour),
	})
	if alarmCollection.Err() != nil {
		log.Panic(alarmCollection.Err())
	}

	// check how many alarms there are before we iterate over them (though this is just an estimate)
	if err := alarmCollection.Preview(); err != nil {
		slog.Error("Failed to get preview of alarms.", "err", err)
	}

	slog.Info("Alarm summary", "total", alarmCollection.TotalCount())

	// iterate over the alarms (paging is done automatically)
	count := 0
	for alarm, err := range alarmCollection.Items() {
		if err != nil {
			slog.Error("Error iterating alarms", "err", err)
			break
		}
		slog.Info("alarm", "id", alarm.ID(), "type", alarm.Type())
		count += 1
		if count > 2002 {
			break
		}
	}

	// Complex task using concurrency and a sequence of actions
	client.SetDebug(true)

	// Step 1: Select managed objects
	moIter := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
		Query: model.NewInventoryQuery().AddFilterEqStr("name", "ci_*").Build(),
		PaginationOptions: pagination.PaginationOptions{
			MaxItems: 10,
		},
	})
	devices := rill.FromSeq(moIter.Seq(), moIter.Err())

	// Step 2: Create an alarm and return it
	createdAlarms := rill.Map(devices, 5, func(device jsonmodels.ManagedObject) (jsonmodels.Alarm, error) {
		slog.Info("Current device", "id", device.ID(), "creationTime", device.CreationTime())
		alarm := client.Alarms.Create(context.Background(), alarms.CreateOptions{
			Source:   managedobjects.DeviceRef(device.ID()),
			Type:     "ci_rill_test",
			Text:     "Test create alarm",
			Severity: "MAJOR",
			Time:     time.Now(),
		})
		return alarm.Data, alarm.Err
	})

	// Step 3: Process the created alarm
	procErrs := rill.ForEach(createdAlarms, 2, func(alarm jsonmodels.Alarm) error {
		slog.Info("Created new alarm", "id", alarm.ID(), "creationTime", alarm.CreationTime())
		return nil
	})

	if procErrs != nil {
		log.Panic(procErrs)
	}

	// Inventory binaries
	exampleCreateBinary(client)
}

func exampleCreateBinary(client *api.Client) error {
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
