package c8y_test

import (
	"context"
	"fmt"
	"testing"

	c8y "github.com/reubenmiller/go-c8y"
)

func TestMeasurementService_GetMeasurementSeries(t *testing.T) {
	client := createTestClient()

	col, _, _ := getDevices(client, "*WEA*", 1)

	sourceID := col.ManagedObjects[0].ID

	dateFrom, dateTo := c8y.GetDateRange("1d")

	_, _, err := client.Measurement.GetMeasurementSeries(context.Background(), &c8y.MeasurementSeriesOptions{
		Source:    sourceID,
		Variables: []string{"nx_WEA_27_Delta.ANA014", "nx_WEA_27_Delta.ANA017"},
		DateFrom:  dateFrom,
		DateTo:    dateTo,
	})

	if err != nil {
		t.Errorf("No error should be returned. want: nil, got: %s", err)
	}

}
func TestMeasurementService_GetMeasurementCollection(t *testing.T) {
	client := createTestClient()

	col, _, _ := getDevices(client, "*WEA*", 1)

	sourceID := col.ManagedObjects[0].ID

	dateFrom, dateTo := c8y.GetDateRange("1d")
	_, resp, _ := client.Measurement.GetMeasurementCollection(context.Background(), &c8y.MeasurementCollectionOptions{
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Source:   sourceID,
	})

	if resp == nil {
		t.Errorf("Result should not be nil")
	}

	fmt.Printf("json result: %s\n", *resp.JSONData)

	totalmeasurements := resp.JSON.Get("measurements.#").Int()

	if totalmeasurements == 0 {
		t.Errorf("expected more than 0 measurements. want: %d, got: %d", 1, totalmeasurements)
	}
	value := resp.JSON.Get("measurements.0.id")

	if !value.Exists() {
		t.Errorf("expected id to exist. Wanted: Existing but go: no exist")
	}
	fmt.Printf("JSON value: %s", value)
}