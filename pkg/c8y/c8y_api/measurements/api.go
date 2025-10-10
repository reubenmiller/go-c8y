package measurements

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

var ApiMeasurements = "/measurement/measurements"
var ApiMeasurementsSeries = "/measurement/measurements/series"

// Measurement service
type Service core.Service

// ListOptions
type ListOptions struct {
	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// DateFrom Timestamp `url:"dateFrom,omitempty"`
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	Type string `url:"type,omitempty"`

	ValueFragmentType string `url:"valueFragmentType,omitempty"`

	ValueFragmentSeries string `url:"valueFragmentSeries,omitempty"`

	Revert bool `url:"revert,omitempty"`

	// Pagination options
	pagination.PaginationOptions
}

// MeasurementCollectionOptions todo
type DeleteOptions struct {
	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// DateFrom Timestamp `url:"dateFrom,omitempty"`
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`
}

// ListSeriesOptions todo
type ListSeriesOptions struct {
	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	DateFrom string `url:"dateFrom,omitempty"`

	DateTo string `url:"dateTo,omitempty"`

	AggregationType string `url:"aggregationType,omitempty"`

	Variables []string `url:"series,omitempty"`

	Revert bool `url:"revert,omitempty"`
}

// MeasurementCollection is the generic data structure which contains the response cumulocity when requesting a measurement collection
type MeasurementCollection struct {
	*model.BaseResponse

	Measurements []Measurement `json:"measurements"`
}

// Measurements represents multiple measurements
type Measurements struct {
	Measurements []MeasurementRepresentation `json:"measurements"`
}

// GetMeasurements return a measurement collection (multiple measurements)
func (s *Service) List(ctx context.Context, opt *ListOptions) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiMeasurements)
}

// DeleteMeasurements removes a measurement collection
func (s *Service) Delete(ctx context.Context, opt *DeleteOptions) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiMeasurements)
}

// Create posts a new measurement to the platform
func (s *Service) Create(ctx context.Context, body any) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodPost).
		SetContentType("application/json").
		SetBody(body).
		SetURL(ApiMeasurements)
}

// CreateMeasurements posts multiple measurement to the platform
func (s *Service) CreateMultiple(ctx context.Context, body any) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetContentType("application/vnd.com.nsn.cumulocity.measurementCollection+json").
		SetURL(ApiMeasurements)
}

// MeasurementSeriesDefinition represents information about a single series
type MeasurementSeriesDefinition struct {
	Unit string `json:"unit"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// MeasurementSeriesValueGroup represents multiple values for multiple series for a single timestamp
type MeasurementSeriesValueGroup struct {
	Timestamp time.Time `json:"timestamp"`
	Values    []float64 `json:"values"`
}

// MeasurementSeriesAggregateValueGroup represents multiple aggregate values for multiple series for a single timestamp
type MeasurementSeriesAggregateValueGroup struct {
	Timestamp time.Time                   `json:"timestamp"`
	Values    []MeasurementAggregateValue `json:"values"`
}

// MeasurementSeriesGroup represents a group of series values (no aggregate values)
type MeasurementSeriesGroup struct {
	DeviceID  string                        `json:"deviceId"`
	Series    []MeasurementSeriesDefinition `json:"series"`
	Values    []MeasurementSeriesValueGroup `json:"values"`
	DateFrom  time.Time                     `json:"dateFrom"`
	DateTo    time.Time                     `json:"dateTo"`
	Truncated bool                          `json:"truncated"`
}

// MeasurementSeriesAggregateGroup represents a group of aggregate series
type MeasurementSeriesAggregateGroup struct {
	Series    []MeasurementSeriesDefinition          `json:"series"`
	Values    []MeasurementSeriesAggregateValueGroup `json:"values"`
	DateFrom  time.Time                              `json:"dateFrom"`
	DateTo    time.Time                              `json:"dateTo"`
	Truncated bool                                   `json:"truncated"`
}

// MeasurementAggregateValue represents the aggregate value of a single measurement.
type MeasurementAggregateValue struct {
	Min model.Number `json:"min"`
	Max model.Number `json:"max"`
}

// UnmarshalJSON converts the Cumulocity measurement Series response to a format which is easier to parse.
//
//	{
//	    "series": [ "c8y_Temperature.A", "c8y_Temperature.B" ],
//	    "unit": [ "degC", "degC" ],
//	    "truncated": true,
//	    "values": [
//	        { "timestamp": "2018-11-11T23:20:00.000+01:00", values: [0.0001, 0.1001] },
//	        { "timestamp": "2018-11-11T23:20:01.000+01:00", values: [0.1234, 2.2919] },
//	        { "timestamp": "2018-11-11T23:20:02.000+01:00", values: [0.8370, 4.8756] }
//	    ]
//	}
func (d *MeasurementSeriesGroup) UnmarshalJSON(data []byte) error {
	c8ySeries := gjson.ParseBytes(data)

	// Get the series definitions
	var seriesDefinitions []MeasurementSeriesDefinition

	c8ySeries.Get("series").ForEach(func(_, item gjson.Result) bool {
		v := &MeasurementSeriesDefinition{}
		if err := json.Unmarshal([]byte(item.String()), &v); err != nil {
			slog.Info("Could not unmarshal series definition", "value", item.String())
		}

		seriesDefinitions = append(seriesDefinitions, *v)
		return true
	})

	d.Series = seriesDefinitions
	d.Truncated = c8ySeries.Get("truncated").Bool()

	totalSeries := len(seriesDefinitions)

	// Get each series values
	var allSeries []MeasurementSeriesValueGroup
	c8ySeries.Get("values").ForEach(func(key, values gjson.Result) bool {
		timestamp, err := time.Parse(time.RFC3339, key.Str)

		if err != nil {
			panic(fmt.Sprintf("Invalid timestamp: %s", key.Str))
		}

		seriesValues := &MeasurementSeriesValueGroup{
			Timestamp: timestamp,
			Values:    make([]float64, totalSeries),
		}

		index := 0
		values.ForEach(func(_, value gjson.Result) bool {
			// Note: min and max values are the same when no aggregation is being used!
			// so technically we could get the value from either min or max.
			seriesValues.Values[index] = value.Get("max").Float()
			index++
			return true
		})

		allSeries = append(allSeries, *seriesValues)
		return true
	})

	// Store the first and last timestamps
	if len(allSeries) > 0 {
		d.DateFrom = allSeries[0].Timestamp
		d.DateTo = allSeries[len(allSeries)-1].Timestamp
	}

	d.Values = allSeries
	return nil
}

// UnmarshalJSON controls the conversion of json bytes to the MeasurementSeriesAggregateGroup struct
func (d *MeasurementSeriesAggregateGroup) UnmarshalJSON(data []byte) error {
	c8ySeries := gjson.ParseBytes(data)

	// Get the series definitions
	var seriesDefinitions []MeasurementSeriesDefinition

	c8ySeries.Get("series").ForEach(func(_, item gjson.Result) bool {
		v := &MeasurementSeriesDefinition{}
		if err := json.Unmarshal([]byte(item.String()), &v); err != nil {
			slog.Info("Could not unmarshal series definition", "value", item.String())
		}

		seriesDefinitions = append(seriesDefinitions, *v)
		return true
	})

	d.Series = seriesDefinitions
	d.Truncated = c8ySeries.Get("truncated").Bool()

	totalSeries := len(seriesDefinitions)

	// Get each series values
	var allSeries []MeasurementSeriesAggregateValueGroup
	c8ySeries.Get("values").ForEach(func(key, values gjson.Result) bool {

		slog.Info("Series", "key", key, "values", values)

		timestamp, err := time.Parse(time.RFC3339, key.Str)

		if err != nil {
			panic(fmt.Sprintf("Invalid timestamp: %s", key.Str))
		}

		seriesValues := &MeasurementSeriesAggregateValueGroup{
			Timestamp: timestamp,
			Values:    make([]MeasurementAggregateValue, totalSeries),
		}

		index := 0
		values.ForEach(func(_, value gjson.Result) bool {
			slog.Info("Current value", "value", value)
			v := &MeasurementAggregateValue{}
			json.Unmarshal([]byte(value.String()), &v)

			slog.Info("Full Value", "value", v)

			seriesValues.Values[index] = *v
			index++
			return true
		})

		allSeries = append(allSeries, *seriesValues)
		return true
	})

	// Store the first and last timestamps
	if len(allSeries) > 0 {
		d.DateFrom = allSeries[0].Timestamp
		d.DateTo = allSeries[len(allSeries)-1].Timestamp
	}

	d.Values = allSeries
	return nil
}

// GetMeasurementSeries returns the measurement series for a given source and variables
// The data is returned in a user friendly format to make it easier to use the data
func (s *Service) ListSeries(ctx context.Context, opt *ListSeriesOptions) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiMeasurementsSeries)
}

// MarshalCSV converts the measurement series group to a csv output so it can be more easily parsed by other languages
// Example output
// timestamp,c8y_Temperature.A,c8y_Temperature.B
// 2018-11-23T00:45:39+01:00,60.699993,44.300003
// 2018-11-23T01:45:39+01:00,67.63333,47.199997
func (d *MeasurementSeriesGroup) MarshalCSV(delimiter string) ([]byte, error) {

	useDelimiter := delimiter
	if useDelimiter == "" {
		useDelimiter = ","
	}

	totalSeries := len(d.Series)

	// First column is the timestamp
	headers := make([]string, totalSeries+1)
	row := make([]string, totalSeries+1)

	headers[0] = "timestamp"

	var output string

	for i, header := range d.Series {
		headers[i+1] = fmt.Sprintf("%s.%s", header.Type, header.Name)
		output = strings.Join(headers, useDelimiter) + "\n"
	}

	for _, datapoint := range d.Values {
		row[0] = datapoint.Timestamp.Format(time.RFC3339)
		for i := 0; i < totalSeries; i++ {
			row[i+1] = fmt.Sprintf("%f", datapoint.Values[i])
		}
		output += strings.Join(row, useDelimiter) + "\n"
	}

	return []byte(output), nil
}
