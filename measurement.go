package c8y

import (
	"context"
	"fmt"

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
