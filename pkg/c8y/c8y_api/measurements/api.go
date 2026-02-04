package measurements

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/internal/context"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiMeasurements = "/measurement/measurements"
var ApiMeasurementsSeries = "/measurement/measurements/series"

const ResultProperty = "measurements"

// Measurement service
type Service struct {
	core.Service
	DeviceResolver *managedobjects.DeviceResolver
}

// NewService creates a new measurement service with device resolution capabilities
func NewService(common *core.Service, moService *managedobjects.Service) *Service {
	return &Service{
		Service:        *common,
		DeviceResolver: managedobjects.NewDeviceResolver(moService),
	}
}

// ListOptions
type ListOptions struct {
	// Source device to filter measurements by.
	// Supports resolver strings: direct ID, "name:deviceName", "ext:type:id", "query:..."
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
type MeasurementIterator = pagination.Iterator[jsonmodels.Measurement]

// GetMeasurements return a measurement collection (multiple measurements)
// The Source field supports resolver strings:
//   - "12345" - direct ID
//   - "name:deviceName" - lookup by device name
//   - "ext:c8y_Serial:ABC123" - lookup by external ID
//   - "query:type eq 'c8y_Device'" - lookup by inventory query
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Measurement] {
	// Resolve Source if it contains a resolver scheme
	if opt.Source != "" && s.DeviceResolver != nil {
		// Use resolution context that bypasses deferred execution
		resolutionCtx := ctx
		if ctxhelpers.IsDeferredExecution(ctx) {
			resolutionCtx = context.Background()
		}

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, nil)
		if err != nil {
			return op.Failed[jsonmodels.Measurement](err, true)
		}
		opt.Source = resolvedID
	}

	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewMeasurement)
}

// ListAll returns an iterator for all measurements
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *MeasurementIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Measurement] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewMeasurement,
	)
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
	// Resolve Source if it contains a resolver scheme
	if opt.Source != "" && s.DeviceResolver != nil {
		resolutionCtx := ctx
		if ctxhelpers.IsDeferredExecution(ctx) {
			resolutionCtx = context.Background()
		}

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, nil)
		if err != nil {
			return op.Failed[jsonmodels.Measurement](err, true)
		}
		opt.Source = resolvedID
	}

	return core.Execute(ctx, s.deleteListB(opt), jsonmodels.NewMeasurement)
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
	return core.Execute(ctx, s.createB(body), jsonmodels.NewMeasurement)
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
	return core.Execute(ctx, s.createB(body), jsonmodels.NewMeasurement)
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
	// Source device to filter measurements by.
	// Supports resolver strings: direct ID, "name:deviceName", "ext:type:id", "query:..."
	Source string `url:"source,omitempty"`

	DateFrom string `url:"dateFrom,omitempty"`

	DateTo string `url:"dateTo,omitempty"`

	AggregationFunction []string `url:"aggregationFunction,omitempty"`

	AggregationInterval string `url:"aggregationInterval,omitempty"`

	AggregationType string `url:"aggregationType,omitempty"`

	Variables []string `url:"series,omitempty"`

	Revert bool `url:"revert,omitempty"`
}

// ListSeries returns measurement series for a given source and variables.
// The response includes helper methods to transform the column-based API data
// into a row-based tabular format that's easier to work with.
//
// The Source field supports resolver strings:
//   - "12345" - direct ID
//   - "name:deviceName" - lookup by device name
//   - "ext:c8y_Serial:ABC123" - lookup by external ID
//   - "query:type eq 'c8y_Device'" - lookup by inventory query
func (s *Service) ListSeries(ctx context.Context, opt ListSeriesOptions) op.Result[jsonmodels.MeasurementSeries] {
	var deviceID, deviceName string

	// Resolve Source if it contains a resolver scheme
	if opt.Source != "" && s.DeviceResolver != nil {
		resolutionCtx := ctx
		if ctxhelpers.IsDeferredExecution(ctx) {
			resolutionCtx = context.Background()
		}

		meta := make(map[string]any)
		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, meta)
		if err != nil {
			return op.Failed[jsonmodels.MeasurementSeries](err, true)
		}
		opt.Source = resolvedID
		deviceID = resolvedID

		// Extract device name from metadata if available
		if name, ok := meta["name"].(string); ok {
			deviceName = name
		}
	} else {
		// Direct ID provided
		deviceID = opt.Source
	}

	// Execute request and add device info to result
	result := core.Execute(ctx, s.listSeriesB(opt), jsonmodels.NewMeasurementSeries)
	if result.Err == nil {
		// Populate device info in the measurement series
		result.Data.DeviceID = deviceID
		result.Data.DeviceName = deviceName
	}
	return result
}

func (s *Service) listSeriesB(opt ListSeriesOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiMeasurementsSeries)
	return core.NewTryRequest(s.Client, req)
}
