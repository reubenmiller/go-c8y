package c8y

import (
	"context"
	"fmt"
	"time"
	"encoding/json"

	"github.com/tidwall/gjson"
)

// MeasurementService does something
type MeasurementService service

// MeasurementCollectionOptions todo
type MeasurementCollectionOptions struct {
	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// DateFrom Timestamp `url:"dateFrom,omitempty"`
	DateFrom string `url:"dateFrom,omitempty"`

	DateTo string `url:"dateTo,omitempty"`

	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	ValueFragmentType string `url:"valueFragmentType,omitempty"`

	ValueFragmentSeries string `url:"valueFragmentSeries,omitempty"`

	Revert bool `url:"revert,omitempty"`

	// Pagination options
	PaginationOptions
}

// MeasurementSeriesOptions todo
type MeasurementSeriesOptions struct {
	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	DateFrom string `url:"dateFrom,omitempty"`

	DateTo string `url:"dateTo,omitempty"`

	Variables []string `url:"series,omitempty"`

	Revert bool `url:"revert,omitempty"`
}

// MeasurementCollection is the generic data structure which contains the response cumulocity when requesting a measurement collection
type MeasurementCollection struct {
	*BaseResponse

	Measurements []MeasurementObject `json:"measurements"`

	Items []gjson.Result
}

// GetMeasurementCollection return a measurement collection (multiple measurements)
func (s *MeasurementService) GetMeasurementCollection(ctx context.Context, opt *MeasurementCollectionOptions) (*MeasurementCollection, *Response, error) {
	u := fmt.Sprintf("measurement/measurements")

	queryParams, err := addOptions("", opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, queryParams, nil)
	if err != nil {
		return nil, nil, err
	}

	data := new(MeasurementCollection)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	data.Items = resp.JSON.Get("measurements").Array()

	return data, resp, nil
}

type MeasurementSeries struct {
	Unit string `json:"unit"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type MeasurementSeriesValue struct {
	Timestamp time.Time
	Min float64	`json:"min,omitempty"`
	Max float64	`json:"max,omitempty"`
}

type MeasurementValues struct {
	MeasurementSeries
	
}

type MeasurementValue struct {
	Timestamp time.Time `json:"timestamp"`
	Min float64	`json:"min"`
	Max float64	`json:"max"`
}

type MeasurementSeriesResponse struct {
	Values MeasurementRawValues `json:"values"`	// Needs to be processed more 
	Series []MeasurementSeries	`json:""`
	Truncated bool `json:"truncated"`
}

type MeasurementRawValues struct {
	Values []MeasurementSeriesValue
}

// UnmarshalJSON converts the Cumulocity measurement Series response to a format which is easier to parse.
//
//
// [{
//     "series": [{
//         "type": "myType",
//         "name": "Variable1",
//         "unit": "",
//         "values": [
//             { "timestamp": "2018-11-11T23:20:00.000+01:00", "min": 0.0001, "max": 10.1234 },
//             { "timestamp": "2018-11-11T23:20:00.000+01:00", "min": 1.0001, "max": 9.1234 },
//             { "timestamp": "2018-11-11T23:20:00.000+01:00", "min": -1.123, "max": 50.5 }
//         ]
//     }]
// }]
//
func (d *MeasurementSeriesResponse) UnmarshalJSON(data []byte) error {
	c8ySeries := gjson.ParseBytes(data)

	var allSeries [][]MeasurementSeriesValue

	c8ySeries.ForEach(func(key, values gjson.Result) bool {

		var seriesValues []MeasurementSeriesValue

		timestamp, err := time.Parse(time.RFC3339, key.Str)

		if err != nil {
			panic(fmt.Sprintf("Invalid timestamp: %s", key.Str))
		}
		values.ForEach(func(_, value gjson.Result) bool {
			v := MeasurementSeriesValue{}
			json.Unmarshal([]byte(value.Str), &v)
			v.Timestamp = timestamp

			seriesValues = append(seriesValues, v)
			return true
		})

		allSeries = append(allSeries, seriesValues)
		return true
	})

	d = &allSeries
	return nil
}

// GetMeasurementSeries returns the measurement series for a given source and variables
func (s *MeasurementService) GetMeasurementSeries(ctx context.Context, opt *MeasurementSeriesOptions) (*MeasurementCollection, *Response, error) {
	u := fmt.Sprintf("measurement/measurements")

	queryParams, err := addOptions("", opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, queryParams, nil)
	if err != nil {
		return nil, nil, err
	}

	data := new(MeasurementCollection)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	series := resp.JSON.Get("series").Array()

	totalSeries := len(series)

	values := resp.JSON.Get("values").ForEach(func (key, values gjson.Result) bool {

		// Loop through results
		t, err := time.Parse(time.RFC3339, key.String())


		values.ForEach(func(index, value gjson.Result){
			v := MeasurementSeriesValue{}

			gjson.Unmarshal([]byte(value.Str), &v)
			return true
		})

		seriesValue
		json.Unmarshal()
		return true
	})

	series.

	// values.
	truncated := resp.JSON.Get("truncated").Bool()

	return data, resp, nil
}

// GetMeasurement returns a single measurement
func (s *MeasurementService) GetMeasurement(ctx context.Context, ID string) (*MeasurementObject, *Response, error) {
	u := fmt.Sprintf("measurement/measurements/%s", ID)

	req, err := s.client.NewRequest("GET", u, "", nil)
	if err != nil {
		return nil, nil, err
	}

	data := new(MeasurementObject)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	data.Item = *resp.JSON

	return data, resp, nil
}
