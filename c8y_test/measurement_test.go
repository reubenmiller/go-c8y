package c8y_test

import (
	"context"
	"fmt"
	"testing"

	c8y "github.com/reubenmiller/go-c8y"
)

func TestMeasurementService_GetMeasurementCollection(t *testing.T) {
	client := createTestClient()
	dateFrom, dateTo := c8y.GetDateRange("1d")
	_, resp, _ := client.Measurement.GetMeasurementCollection(context.Background(), &c8y.MeasurementCollectionOptions{
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Source:   "31032",
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
