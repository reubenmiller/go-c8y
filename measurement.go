package c8y

import (
	"context"
	"fmt"
	"log"
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
}

// GetMeasurementCollection return the measurement collection
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

	mcol := new(MeasurementCollection)

	resp, err := s.client.Do(ctx, req, mcol)
	if err != nil {
		return nil, resp, err
	}

	log.Printf("Total count: %d\n", len(mcol.Measurements))
	if len(mcol.Measurements) > 0 {
		log.Printf("Last time: %v\n", mcol.Measurements[0].Time)
	}
	log.Printf("Measurement Collection: currentPage=%d, pageSize=%v\n", *mcol.Statistics.CurrentPage, *mcol.Statistics.PageSize)

	return mcol, resp, nil
}
