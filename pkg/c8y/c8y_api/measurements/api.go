package measurements

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiMeasurements = "/measurement/measurements"
var ApiMeasurementsSeries = "/measurement/measurements/series"

const ResultProperty = "measurements"

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

// GetMeasurements return a measurement collection (multiple measurements)
func (s *Service) List(ctx context.Context, opt ListOptions) (*model.MeasurementRepresentationCollection, error) {
	return core.ExecuteResultOnly[model.MeasurementRepresentationCollection](ctx, s.ListB(opt))
}

func (s *Service) ListB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiMeasurements)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// DeleteListOptions to control which measurements are to be deleted
type DeleteListOptions struct {
	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// DateFrom Timestamp `url:"dateFrom,omitempty"`
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`
}

// DeleteList removes a collection of measurements
func (s *Service) DeleteList(ctx context.Context, opt DeleteListOptions) error {
	return core.ExecuteNoResult(ctx, s.DeleteListB(opt))
}

func (s *Service) DeleteListB(opt DeleteListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiMeasurements)
	return core.NewTryRequest(s.Client, req)
}

// Create posts a new measurement to the platform
func (s *Service) Create(ctx context.Context, body any) (*model.Measurement, error) {
	return core.ExecuteResultOnly[model.Measurement](ctx, s.CreateB(body))
}

func (s *Service) CreateB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetContentType(types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiMeasurements)
	return core.NewTryRequest(s.Client, req)
}

// CreateList creates multiple measurements to the platform
func (s *Service) CreateList(ctx context.Context, body any) (*model.MeasurementCollection, error) {
	return core.ExecuteResultOnly[model.MeasurementCollection](ctx, s.CreateB(body))
}

func (s *Service) CreateListB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetContentType(types.MimeTypeCumulocityMeasurementCollection).
		SetBody(body).
		SetURL(ApiMeasurements)
	return core.NewTryRequest(s.Client, req)
}

// ListSeriesOptions todo
type ListSeriesOptions struct {
	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	DateFrom string `url:"dateFrom,omitempty"`

	DateTo string `url:"dateTo,omitempty"`

	AggregationFunction []string `url:"aggregationFunction,omitempty"`

	AggregationInterval string `url:"aggregationInterval,omitempty"`

	AggregationType string `url:"aggregationType,omitempty"`

	Variables []string `url:"series,omitempty"`

	Revert bool `url:"revert,omitempty"`
}

// GetMeasurementSeries returns the measurement series for a given source and variables
// The data is returned in a user friendly format to make it easier to use the data
func (s *Service) ListSeries(ctx context.Context, opt *ListSeriesOptions) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiMeasurementsSeries)
}
