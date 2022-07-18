package c8y_test

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func measurmentFactory(client *c8y.Client, deviceID string, valueFragmentType, ValueFragmentSeries string) func(float64) (*c8y.Measurement, *c8y.Response, error) {
	counter := 1
	return func(value float64) (*c8y.Measurement, *c8y.Response, error) {
		counter++
		measValue, _ := c8y.NewSimpleMeasurementRepresentation(c8y.SimpleMeasurementOptions{
			SourceID:            deviceID,
			Type:                "TestSeries1",
			ValueFragmentType:   valueFragmentType,
			ValueFragmentSeries: ValueFragmentSeries,
			Unit:                "Counter",
			Value:               value,
		})
		return client.Measurement.Create(
			context.Background(),
			*measValue,
		)
	}
}

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
func TestMeasurementService_GetMeasurements(t *testing.T) {
	client := createTestClient()

	testDevice, _, err := client.Inventory.CreateDevice(context.Background(), "TestDevice")
	testingutils.Ok(t, err)

	// Create a test measurement
	measurement1, _ := c8y.NewSimpleMeasurementRepresentation(c8y.SimpleMeasurementOptions{
		SourceID:            testDevice.ID,
		Type:                "TestSeries1",
		ValueFragmentType:   "nx_TestDevice",
		ValueFragmentSeries: "Series1",
		Unit:                "Counter",
	})
	meas, _, err := client.Measurement.Create(context.Background(), *measurement1)
	testingutils.Ok(t, err)
	testingutils.Assert(t, meas != nil, "Measurement shouldn't be nil")

	defer client.Inventory.Delete(context.Background(), testDevice.ID)

	// Get a list of the measurements
	dateFrom, _ := c8y.GetDateRange("1d")

	measCollection, resp, _ := client.Measurement.GetMeasurements(context.Background(), &c8y.MeasurementCollectionOptions{
		DateFrom: dateFrom,
		Source:   testDevice.ID,
	})

	if resp == nil {
		t.Errorf("Result should not be nil")
	}

	testingutils.Equals(t, 1, len(measCollection.Items))

	if resp != nil {
		log.Printf("json result: %s\n", resp.Body())

		totalmeasurements := resp.JSON("measurements.#").Int()

		if totalmeasurements != 1 {
			t.Errorf("expected more than 0 measurements. want: %d, got: %d", 1, totalmeasurements)
		}
		value := resp.JSON("measurements.0.id")

		if !value.Exists() {
			t.Errorf("expected id to exist. Wanted: Existing but go: no exist")
		}
		log.Printf("JSON value: %s", value)
	}
}

// TestMeasurementService_MarshalMeasurement tests the custom json marshalling function for the
// measurement representation.
func TestMeasurementService_MarshalMeasurement(t *testing.T) {
	timestamp, _ := time.Parse(time.RFC3339, "2018-11-25T14:41:51+01:00")

	m, err := c8y.NewSimpleMeasurementRepresentation(c8y.SimpleMeasurementOptions{
		SourceID:            "12345",
		Timestamp:           &timestamp,
		Type:                "TestSeries1",
		ValueFragmentType:   "c8y_Temperature",
		ValueFragmentSeries: "A",
		Value:               1.101,
		Unit:                "degC",
		FragmentType:        []string{"c8y_Test"},
	})
	testingutils.Ok(t, err)

	mJSON, err := json.Marshal(m)

	if err != nil {
		t.Errorf("Decoding threw an error when marshalling measurement to json. wanted: nil, got: %s", err)
	}

	expectedOutput := `{"source":{"id":"12345"},"type":"TestSeries1","time":"2018-11-25T14:41:51+01:00","c8y_Test":{},"c8y_Temperature":{"A":{"value":1.101,"unit":"degC"}}}`

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
			{
				Name: "c8y_Temperature",
				Values: []c8y.ValueFragmentSeries{
					{
						Name:  "A",
						Value: 1.101,
						Unit:  "degC",
					},
					{
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
	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)

	m, _ := c8y.NewSimpleMeasurementRepresentation(c8y.SimpleMeasurementOptions{
		SourceID:            testDevice.ID,
		Timestamp:           nil,
		Type:                "c8yTest",
		ValueFragmentType:   "c8y_Temperature",
		ValueFragmentSeries: "A",
		Value:               1.101,
		Unit:                "degC",
		FragmentType:        []string{"c8y_Test"},
	})

	data, resp, err := client.Measurement.Create(context.Background(), *m)

	if resp.StatusCode() != 201 {
		t.Errorf("Unexpected server return code. wanted: 201, got: %d", resp.StatusCode())
	}

	if err != nil {
		t.Errorf("Unexpected error when creating measurement. wanted: nil, got: %s", err)
	}

	if data != nil {
		log.Printf("measurement: %s\n", data.Item.String())
	}
}

func TestMeasurementService_CreateWithDifferentTypes(t *testing.T) {
	client := createTestClient()
	device, _ := createRandomTestDevice()

	createMeasurement := func(value interface{}) *c8y.Measurement {
		m, _ := c8y.NewSimpleMeasurementRepresentation(c8y.SimpleMeasurementOptions{
			SourceID:            device.ID,
			Timestamp:           nil,
			Type:                "c8yTest",
			ValueFragmentType:   "c8y_Temperature",
			ValueFragmentSeries: "A",
			Value:               value,
			Unit:                "degC",
			FragmentType:        []string{"c8y_Test"},
		})

		data, resp, err := client.Measurement.Create(context.Background(), *m)

		if resp.StatusCode() != 201 {
			t.Errorf("Unexpected server return code. wanted: 201, got: %d", resp.StatusCode())
		}
		if err != nil {
			t.Errorf("Unexpected error when creating measurement. wanted: nil, got: %s", err)
		}
		if data != nil {
			log.Printf("measurement: %s\n", data.Item.Raw)
		}

		path := "c8y_Temperature.A.value"

		switch v := value.(type) {
		case []byte:
			if actualVal := data.Item.Get(path).String(); string(v) != string(actualVal) {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		case []rune:
			if actualVal := data.Item.Get(path).String(); string(v) != string(actualVal) {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		case string:
			if actualVal := data.Item.Get(path).String(); v != string(actualVal) {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		case bool:
			if actualVal := data.Item.Get(path).Bool(); v != actualVal {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		case int:
			if actualVal := data.Item.Get(path).Int(); int64(v) != actualVal {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		case int8:
			if actualVal := data.Item.Get(path).Int(); int64(v) != actualVal {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		case int16:
			if actualVal := data.Item.Get(path).Int(); int64(v) != actualVal {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		case int32:
			if actualVal := data.Item.Get(path).Int(); int64(v) != actualVal {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		case int64:
			if actualVal := data.Item.Get(path).Int(); v != actualVal {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		case uint:
			if actualVal := data.Item.Get(path).Uint(); uint64(v) != actualVal {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		case uint8:
			if actualVal := data.Item.Get(path).Uint(); uint64(v) != actualVal {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		case uint16:
			if actualVal := data.Item.Get(path).Uint(); uint64(v) != actualVal {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		case uint32:
			if actualVal := data.Item.Get(path).Uint(); uint64(v) != actualVal {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		case uint64:
			if actualVal := data.Item.Get(path).Uint(); v != actualVal {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		case float32:
			if actualVal := data.Item.Get(path).Float(); !testingutils.AlmostEqual(float64(v), actualVal) {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		case float64:
			if actualVal := data.Item.Get(path).Float(); !testingutils.AlmostEqual(float64(v), actualVal) {
				t.Errorf("Invalid value. wanted: %v, got: %v", v, actualVal)
			}
		default:
			t.Errorf("Unsupported data type")
		}

		return data
	}

	// Float values
	createMeasurement(float64(1.64))
	createMeasurement(float32(1.32))

	// integer values
	createMeasurement(int64(64))
	createMeasurement(int32(32))
	createMeasurement(int16(16))
	createMeasurement(int8(8))
	createMeasurement(int(101))

	// unsigned integer values
	createMeasurement(uint64(64))
	createMeasurement(uint32(32))
	createMeasurement(uint16(16))
	createMeasurement(uint8(8))
	createMeasurement(uint(101))

	// boolean values (No longer supported by c8y)
	// createMeasurement(true)
	// createMeasurement(false)
}

func TestMeasurementService_GetMeasurement_DeleteMeasurement(t *testing.T) {
	client := createTestClient()
	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)

	// Create a test measurement
	measurement1, _ := c8y.NewSimpleMeasurementRepresentation(c8y.SimpleMeasurementOptions{
		SourceID:            testDevice.ID,
		Type:                "TestSeries1",
		ValueFragmentType:   "nx_TestDevice",
		ValueFragmentSeries: "Series1",
		Unit:                "Counter",
	})
	meas, _, err := client.Measurement.Create(context.Background(), *measurement1)
	testingutils.Ok(t, err)
	testingutils.Assert(t, meas != nil, "Measurement shouldn't be nil")

	if meas == nil {
		return
	}

	meas2, resp, err := client.Measurement.GetMeasurement(
		context.Background(),
		meas.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, meas.ID, meas2.ID)

	// Remove measurement
	resp, err = client.Measurement.Delete(
		context.Background(),
		meas2.ID,
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())

	// Check if the measurement has been deleted
	meas3, resp, err := client.Measurement.GetMeasurement(
		context.Background(),
		meas2.ID,
	)
	testingutils.Assert(t, err != nil, "Error should not be nil")
	testingutils.Equals(t, http.StatusNotFound, resp.StatusCode())
	testingutils.Assert(t, meas3.ID == "", "ID should be empty when the object is not found")
}

func TestMeasurementService_DeleteMeasurements(t *testing.T) {
	client := createTestClient()
	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)

	valueFragmentType := "nx_Type1"
	createMeasVariable1 := measurmentFactory(client, testDevice.ID, valueFragmentType, "Variable1")
	createMeasVariable2 := measurmentFactory(client, testDevice.ID, valueFragmentType, "Variable2")

	meas1, resp, err := createMeasVariable1(1.0)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Assert(t, meas1.ID != "", "ID should not be empty")

	meas2, resp, err := createMeasVariable2(2.0)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Assert(t, meas2.ID != "", "ID should not be empty")

	searchOptions := &c8y.MeasurementCollectionOptions{
		Source:            testDevice.ID,
		ValueFragmentType: valueFragmentType,
	}

	measCol1, resp, err := client.Measurement.GetMeasurements(
		context.Background(),
		searchOptions,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, 2, len(measCol1.Measurements))

	// Delete the measurements
	resp, err = client.Measurement.DeleteMeasurements(
		context.Background(),
		searchOptions,
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())

	// Check that the measurements have been removed
	deletedMeas1, resp, err := client.Measurement.GetMeasurement(
		context.Background(),
		meas1.ID,
	)
	testingutils.Assert(t, err != nil, "Error should not be nil")
	testingutils.Equals(t, http.StatusNotFound, resp.StatusCode())
	testingutils.Equals(t, "", deletedMeas1.ID)

	// Check that the measurements have been removed
	deletedMeas2, resp, err := client.Measurement.GetMeasurement(
		context.Background(),
		meas1.ID,
	)
	testingutils.Assert(t, err != nil, "Error should not be nil")
	testingutils.Equals(t, http.StatusNotFound, resp.StatusCode())
	testingutils.Equals(t, "", deletedMeas2.ID)
}

func TestMeasurementService_CreateMeasurements(t *testing.T) {
	client := createTestClient()
	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)

	valueFragmentType := "nx_common"

	meas1Value, _ := c8y.NewSimpleMeasurementRepresentation(c8y.SimpleMeasurementOptions{
		SourceID:            testDevice.ID,
		Type:                "TestSeries1",
		ValueFragmentType:   valueFragmentType,
		ValueFragmentSeries: "Signal1",
		Unit:                "Counter",
		Value:               1.1,
	})

	meas2Value, _ := c8y.NewSimpleMeasurementRepresentation(c8y.SimpleMeasurementOptions{
		SourceID:            testDevice.ID,
		Type:                "TestSeries2",
		ValueFragmentType:   valueFragmentType,
		ValueFragmentSeries: "Signal2",
		Unit:                "Counter",
		Value:               2.0,
	})

	measValues := &c8y.Measurements{
		Measurements: []c8y.MeasurementRepresentation{
			*meas1Value,
			*meas2Value,
		},
	}

	measurements, resp, err := client.Measurement.CreateMeasurements(
		context.Background(),
		measValues,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Equals(t, 2, len(measurements.Measurements))
}
