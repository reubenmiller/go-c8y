package c8y_test

import (
	"context"
	"encoding/json"
	"log"
	"testing"
	"time"

	c8y "github.com/reubenmiller/go-c8y"
)

func TestMeasurementService_GetMeasurementSeries(t *testing.T) {
	client := createTestClient()

	col, _, _ := getDevices(client, "*", 1)

	sourceID := col.ManagedObjects[0].ID

	dateFrom, dateTo := c8y.GetDateRange("2d")

	data, resp, err := client.Measurement.GetMeasurementSeries(context.Background(), &c8y.MeasurementSeriesOptions{
		Source:    sourceID,
		Variables: []string{"c8y_Temperature.A", "c8y_Temperature.B"},
		DateFrom:  dateFrom,
		DateTo:    dateTo,
	})

	if err != nil {
		t.Errorf("No error should be returned. want: nil, got: %s", err)
	}

	csv, _ := data.MarshalCSV(",")

	log.Printf("csv: %s\n", csv)

	if respJson, err := json.Marshal(data); err != nil {
		t.Errorf("Could not convert object to json. %v", data)
	} else {
		log.Printf("JSON Response: %s\n", respJson)
	}

	log.Printf("Response: %v\n", resp)
}
func TestMeasurementService_GetMeasurementCollection(t *testing.T) {
	client := createTestClient()

	col, _, _ := getDevices(client, "*", 1)

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

	log.Printf("json result: %s\n", *resp.JSONData)

	totalmeasurements := resp.JSON.Get("measurements.#").Int()

	if totalmeasurements == 0 {
		t.Errorf("expected more than 0 measurements. want: %d, got: %d", 1, totalmeasurements)
	}
	value := resp.JSON.Get("measurements.0.id")

	if !value.Exists() {
		t.Errorf("expected id to exist. Wanted: Existing but go: no exist")
	}
	log.Printf("JSON value: %s", value)
}

// TestMeasurementService_MarshalMeasurement tests the custom json marshalling function for the
// measurement representation.
func TestMeasurementService_MarshalMeasurement(t *testing.T) {
	timestamp, _ := time.Parse(time.RFC3339, "2018-11-25T14:41:51+01:00")

	m, _ := c8y.NewSimpleMeasurementRepresentation(c8y.SimpleMeasurementOptions{
		SourceID:            "12345",
		Timestamp:           &timestamp,
		ValueFragmentType:   "c8y_Temperature",
		ValueFragmentSeries: "A",
		Value:               1.101,
		Unit:                "degC",
		FragmentType:        []string{"c8y_Test"},
	})

	mJSON, err := json.Marshal(m)

	if err != nil {
		t.Errorf("Decoding threw an error when marshalling measurement to json. wanted: nil, got: %s", err)
	}

	expectedOutput := `{"source":{"id":"12345"},"time":"2018-11-25T14:41:51+01:00","c8y_Test":{},"c8y_Temperature":{"A":{"value":1.101,"unit":"degC"}}}`

	if string(mJSON) != expectedOutput {
		t.Errorf("json does not match. wanted: %s, got: %s", expectedOutput, mJSON)
	}

	log.Printf("json: %s\n", mJSON)
}

func TestMeasurementService_MarshalMeasurementMultipleSeries(t *testing.T) {
	timestamp, _ := time.Parse(time.RFC3339, "2018-11-25T14:41:51+01:00")

	m := c8y.MeasurementRepresentation{
		Source: c8y.MeasurementSource{
			ID: "12345",
		},
		Timestamp: timestamp,
		Fragments: c8y.NewFragmentNameSeries("c8y_Test"),
		ValueFragmentTypes: []c8y.ValueFragmentType{
			c8y.ValueFragmentType{
				Name: "c8y_Temperature",
				Values: []c8y.ValueFragmentSeries{
					c8y.ValueFragmentSeries{
						Name:  "A",
						Value: 1.101,
						Unit:  "degC",
					},
					c8y.ValueFragmentSeries{
						Name:  "B",
						Value: 56.876,
						Unit:  "degC",
					},
				},
			},
		},
	}

	mJSON, err := json.Marshal(m)

	if err != nil {
		t.Errorf("Decoding threw an error when marshalling measurement to json. wanted: nil, got: %s", err)
	}

	expectedOutput := `{"source":{"id":"12345"},"time":"2018-11-25T14:41:51+01:00","c8y_Test":{},"c8y_Temperature":{"A":{"value":1.101,"unit":"degC"},"B":{"value":56.876,"unit":"degC"}}}`

	if string(mJSON) != expectedOutput {
		t.Errorf("json does not match. wanted: %s, got: %s", expectedOutput, mJSON)
	}

	log.Printf("json: %s\n", mJSON)
}

func TestMeasurementService_Create(t *testing.T) {
	client := createTestClient()

	m, _ := c8y.NewSimpleMeasurementRepresentation(c8y.SimpleMeasurementOptions{
		SourceID:            CumulocityConfiguration.ExampleDevice.ID,
		Timestamp:           nil,
		Type:                "c8yTest",
		ValueFragmentType:   "c8y_Temperature",
		ValueFragmentSeries: "A",
		Value:               1.101,
		Unit:                "degC",
		FragmentType:        []string{"c8y_Test"},
	})

	data, resp, err := client.Measurement.Create(context.Background(), *m)

	if resp.StatusCode != 201 {
		t.Errorf("Unexpected server return code. wanted: 201, got: %d", resp.StatusCode)
	}

	if err != nil {
		t.Errorf("Unexpected error when creating measurement. wanted: nil, got: %s", err)
	}

	if data != nil {
		log.Printf("measurement: %s\n", data.Item.String())
	}
}
