package measurements

import (
	"context"
	"iter"
	"log/slog"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
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

// MeasurementIterator provides iteration over measurements
type MeasurementIterator struct {
	items iter.Seq[jsonmodels.Measurement]
	err   error
}

func (it *MeasurementIterator) Items() iter.Seq[jsonmodels.Measurement] {
	return it.items
}

func (it *MeasurementIterator) Err() error {
	return it.err
}

func paginateMeasurements(ctx context.Context, fetch func(page int) op.Result[jsonmodels.Measurement], maxItems int64) *MeasurementIterator {
	iterator := &MeasurementIterator{}

	iterator.items = func(yield func(jsonmodels.Measurement) bool) {
		page := 1
		count := int64(0)
		for {
			result := fetch(page)
			if result.Err != nil {
				iterator.err = result.Err
				return
			}
			countBeforeResults := count
			for doc := range result.Data.Iter() {
				if maxItems > 0 && count >= maxItems {
					return
				}
				item := jsonmodels.NewMeasurement(doc.Bytes())
				if !yield(item) {
					return
				}
				count++
			}
			if countBeforeResults == count {
				slog.Info("Stopping pagination as results array is empty")
				return
			}

			totalPages, ok := result.Meta["totalPages"].(int64)
			if ok && page >= int(totalPages) {
				return
			}
			page++
		}
	}

	return iterator
}

// GetMeasurements return a measurement collection (multiple measurements)
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Measurement] {
	return core.ExecuteReturnCollection(ctx, s.ListB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewMeasurement)
}

// ListAll returns an iterator for all measurements
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *MeasurementIterator {
	if opts.PageSize == 0 {
		opts.PageSize = 2000
	}
	return paginateMeasurements(ctx, func(page int) op.Result[jsonmodels.Measurement] {
		opts.CurrentPage = page
		return s.List(ctx, opts)
	}, opts.GetMaxItems())
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
func (s *Service) DeleteList(ctx context.Context, opt DeleteListOptions) op.Result[jsonmodels.Measurement] {
	return core.ExecuteReturnResult(ctx, s.DeleteListB(opt), jsonmodels.NewMeasurement)
}

func (s *Service) DeleteListB(opt DeleteListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiMeasurements)
	return core.NewTryRequest(s.Client, req)
}

// Create posts a new measurement to the platform
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Measurement] {
	return core.ExecuteReturnResult(ctx, s.CreateB(body), jsonmodels.NewMeasurement)
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
func (s *Service) CreateList(ctx context.Context, body any) op.Result[jsonmodels.Measurement] {
	return core.ExecuteReturnResult(ctx, s.CreateB(body), jsonmodels.NewMeasurement)
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
