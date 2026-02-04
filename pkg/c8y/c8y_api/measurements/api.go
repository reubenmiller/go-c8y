package measurements

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/source"
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

	// SourceRef allows resolving the source from various references (external ID, name, query, etc.)
	// If set, this takes precedence over Source field
	SourceRef source.Resolver `url:"-"`

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

// Resolve resolves all reference fields (SourceRef) to their concrete values.
// Only resolves if the direct field (Source) is not already set.
func (opt *ListOptions) Resolve(ctx context.Context) error {
	if opt.SourceRef != nil && opt.Source == "" {
		result, err := opt.SourceRef.ResolveID(ctx)
		if err != nil {
			return err
		}
		opt.Source = result.ID
	}
	return nil
}

// MeasurementIterator provides iteration over measurements
type MeasurementIterator = pagination.Iterator[jsonmodels.Measurement]

// GetMeasurements return a measurement collection (multiple measurements)
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Measurement] {
	if err := opt.Resolve(ctx); err != nil {
		return op.Failed[jsonmodels.Measurement](err, true)
	}

	return core.ExecuteReturnCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewMeasurement)
}

// ListAll returns an iterator for all measurements
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *MeasurementIterator {
	return pagination.Paginate(ctx, opts.PaginationOptions, func() op.Result[jsonmodels.Measurement] {
		return s.List(ctx, opts)
	}, jsonmodels.NewMeasurement)
}

func (s *Service) listB(opt any) *core.TryRequest {
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
func (s *Service) DeleteList(ctx context.Context, opt DeleteListOptions) op.Result[jsonmodels.Measurement] {
	return core.ExecuteReturnResult(ctx, s.deleteListB(opt), jsonmodels.NewMeasurement)
}

func (s *Service) deleteListB(opt DeleteListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiMeasurements)
	return core.NewTryRequest(s.Client, req)
}

// Create posts a new measurement to the platform
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Measurement] {
	return core.ExecuteReturnResult(ctx, s.createB(body), jsonmodels.NewMeasurement)
}

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetContentType(types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiMeasurements)
	return core.NewTryRequest(s.Client, req)
}

// CreateList creates multiple measurements to the platform
func (s *Service) CreateList(ctx context.Context, body any) op.Result[jsonmodels.Measurement] {
	return core.ExecuteReturnResult(ctx, s.createB(body), jsonmodels.NewMeasurement)
}

func (s *Service) createListB(body any) *core.TryRequest {
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
