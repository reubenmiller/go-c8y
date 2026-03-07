package measurements

import (
	"context"
	"encoding/json"
	"time"

	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/api/contexthelpers"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/realtime"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/pkg/jsonUtilities"
	"resty.dev/v3"
)

var ApiMeasurements = "/measurement/measurements"
var ApiMeasurement = "/measurement/measurements/{id}"
var ApiMeasurementsSeries = "/measurement/measurements/series"

const ParamId = "id"

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
	// Use the typed helpers: managedobjects.ByName, ByExternalID, ByQuery, ByID,
	// or cast a string variable with managedobjects.DeviceRef(id).
	Source managedobjects.DeviceRef `url:"source,omitempty"`

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
		resolutionCtx := ctxhelpers.ResolutionContext(ctx)

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, nil)
		if err != nil {
			return op.Failed[jsonmodels.Measurement](err, true)
		}
		opt.Source = managedobjects.DeviceRef(resolvedID)
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
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiMeasurements)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// DeleteListOptions to control which measurements are to be deleted
type DeleteListOptions struct {
	// Source device to filter measurements by.
	// Use the typed helpers: managedobjects.ByName, ByExternalID, ByQuery, ByID,
	// or cast a string variable with managedobjects.DeviceRef(id).
	Source managedobjects.DeviceRef `url:"source,omitempty"`

	// DateFrom Timestamp `url:"dateFrom,omitempty"`
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`
}

// DeleteList removes a collection of measurements
func (s *Service) DeleteList(ctx context.Context, opt DeleteListOptions) op.Result[core.NoContent] {
	// Resolve Source if it contains a resolver scheme
	if opt.Source != "" && s.DeviceResolver != nil {
		resolutionCtx := ctxhelpers.ResolutionContext(ctx)

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, nil)
		if err != nil {
			return op.Failed[core.NoContent](err, true)
		}
		opt.Source = managedobjects.DeviceRef(resolvedID)
	}

	return core.ExecuteNoContent(ctx, s.deleteListB(opt))
}

func (s *Service) deleteListB(opt DeleteListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiMeasurements)
	return core.NewTryRequest(s.Client, req)
}

// Delete removes a single measurement by ID
func (s *Service) Delete(ctx context.Context, ID string) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(ID))
}

func (s *Service) deleteB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, ID).
		SetURL(ApiMeasurement)
	return core.NewTryRequest(s.Client, req)
}

// CreateOptions for creating a measurement with resolver support
type CreateOptions struct {
	// Source device identifier.
	// Use the typed helpers: managedobjects.ByName, ByExternalID, ByQuery, ByID,
	// or cast a string variable with managedobjects.DeviceRef(id).
	Source managedobjects.DeviceRef

	// Type of the measurement
	Type string

	// Time when the measurement was taken
	Time time.Time

	// AdditionalProperties allows for custom fields to be added to the measurement
	// Can be a struct, map[string]interface{}, or any JSON-serializable type
	// These properties are deep-merged with the base measurement fields
	AdditionalProperties interface{}
}

// Create posts a new measurement to the platform
// Accepts either CreateOptions (for resolver support and property merging) or any other type (passed through as-is)
//
// Using CreateOptions:
//
//	result := client.Measurements.Create(ctx, measurements.CreateOptions{
//	    Source: "name:myDevice",  // Resolver string
//	    Type: "c8y_Temperature",
//	    Time: time.Now(),
//	    AdditionalProperties: map[string]interface{}{
//	        "c8y_Temperature": map[string]interface{}{
//	            "T": map[string]interface{}{"value": 23.5, "unit": "°C"},
//	        },
//	    },
//	})
//
// Using direct struct/map:
//
//	result := client.Measurements.Create(ctx, model.Measurement{...})
//	result := client.Measurements.Create(ctx, map[string]interface{}{...})
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Measurement] {
	// Check if body is CreateOptions - if so, handle resolver and merge logic
	if opts, ok := body.(CreateOptions); ok {
		return s.createWithOptions(ctx, opts)
	}

	// Otherwise, pass through as-is
	return core.Execute(ctx, s.createB(body), jsonmodels.NewMeasurement)
}

// createWithOptions handles the CreateOptions case with resolver support and property merging
func (s *Service) createWithOptions(ctx context.Context, opts CreateOptions) op.Result[jsonmodels.Measurement] {
	// Resolve the source device and capture metadata
	sourceID := string(opts.Source)
	meta := make(map[string]any)

	if sourceID != "" && s.DeviceResolver != nil {
		resolutionCtx := ctxhelpers.ResolutionContext(ctx)

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, managedobjects.DeviceRef(sourceID), meta)
		if err != nil {
			return op.Failed[jsonmodels.Measurement](err, true)
		}
		sourceID = resolvedID

		// Populate metadata with resolved device information
		meta["id"] = resolvedID
	} else if sourceID != "" {
		// Direct ID provided without resolution
		meta["id"] = sourceID
	}

	// Build base measurement from known fields
	baseMeasurement := map[string]interface{}{
		"source": map[string]interface{}{"id": sourceID},
	}
	if opts.Type != "" {
		baseMeasurement["type"] = opts.Type
	}
	if !opts.Time.IsZero() {
		baseMeasurement["time"] = opts.Time
	}

	// Marshal base measurement to JSON
	baseJSON, err := json.Marshal(baseMeasurement)
	if err != nil {
		return op.Failed[jsonmodels.Measurement](err, true)
	}

	// If there are additional properties, merge them with the base
	var finalJSON []byte
	if opts.AdditionalProperties != nil {
		additionalJSON, err := json.Marshal(opts.AdditionalProperties)
		if err != nil {
			return op.Failed[jsonmodels.Measurement](err, true)
		}

		// Deep merge: additional properties override/extend base properties
		finalJSON, err = jsonUtilities.MergePatch(baseJSON, additionalJSON)
		if err != nil {
			return op.Failed[jsonmodels.Measurement](err, true)
		}
	} else {
		finalJSON = baseJSON
	}

	// Create the measurement with the merged JSON and add metadata
	result := core.Execute(ctx, s.createBWithJSON(finalJSON), jsonmodels.NewMeasurement)

	// Add resolver metadata to result
	if result.Meta == nil {
		result.Meta = make(map[string]any)
	}
	for k, v := range meta {
		result.Meta[k] = v
	}

	return result
}

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiMeasurements)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) createBWithJSON(bodyJSON []byte) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(bodyJSON).
		SetURL(ApiMeasurements)
	return core.NewTryRequest(s.Client, req)
}

// CreateList creates multiple measurements to the platform
func (s *Service) CreateList(ctx context.Context, body any) op.Result[jsonmodels.Measurement] {
	return core.ExecuteCollection(ctx, s.createListB(body), ResultProperty, "", jsonmodels.NewMeasurement)
}

func (s *Service) createListB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetContentType(types.MimeTypeCumulocityMeasurementCollection).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiMeasurements)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// ListSeriesOptions todo
type ListSeriesOptions struct {
	// Source device to filter measurements by.
	// Use the typed helpers: managedobjects.ByName, ByExternalID, ByQuery, ByID,
	// or cast a string variable with managedobjects.DeviceRef(id).
	Source managedobjects.DeviceRef `url:"source,omitempty"`

	DateFrom time.Time `url:"dateFrom,omitzero"`

	DateTo time.Time `url:"dateTo,omitzero"`

	AggregationFunction []string `url:"aggregationFunction,omitempty"`

	AggregationInterval string `url:"aggregationInterval,omitempty"`

	AggregationType string `url:"aggregationType,omitempty"`

	Series []string `url:"series,omitempty"`

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
		resolutionCtx := ctxhelpers.ResolutionContext(ctx)

		meta := make(map[string]any)
		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, meta)
		if err != nil {
			return op.Failed[jsonmodels.MeasurementSeries](err, true)
		}
		opt.Source = managedobjects.DeviceRef(resolvedID)
		deviceID = resolvedID

		// Extract device name from metadata if available
		if name, ok := meta["name"].(string); ok {
			deviceName = name
		}
	} else {
		// Direct ID provided
		deviceID = string(opt.Source)
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
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiMeasurementsSeries)
	return core.NewTryRequest(s.Client, req)
}

// MeasurementStream provides an iterator for realtime measurement subscriptions
type MeasurementStream = realtime.Stream[realtime.StreamData[jsonmodels.Measurement]]

// SubscribeStream subscribes to realtime measurements and returns a typed stream iterator.
// The subscription automatically unsubscribes when the context is cancelled or times out.
//
// IMPORTANT: Always call stream.Close() when done, typically via defer.
// This ensures proper cleanup of the realtime subscription.
//
// Recommended pattern:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	streamResult := client.Measurements.SubscribeStream(ctx, deviceID)
//	if streamResult.Err != nil {
//	    return streamResult.Err
//	}
//	stream := streamResult.Data
//	defer stream.Close() // Required for cleanup
//
//	// Using range with Items() for error handling
//	for item, err := range stream.Items() {
//	    if err != nil {
//	        return err
//	    }
//	    log.Printf("Measurement %s: %s", item.Action, item.Data.Type())
//	    if item.Data.Type() == "targetType" {
//	        break
//	    }
//	}
//
//	// Or using range with Seq() (simpler, errors stop iteration)
//	for item := range stream.Seq() {
//	    log.Printf("Measurement %s: %s", item.Action, item.Data.Type())
//	}
//	if err := stream.Err(); err != nil {
//	    return err
//	}
func (s *Service) SubscribeStream(ctx context.Context, ID string) op.Result[*MeasurementStream] {
	err := s.RealtimeClient.Connect()
	if err != nil {
		return op.Failed[*MeasurementStream](err, false)
	}

	messages := make(chan *realtime.Message, 10)
	pattern := realtime.Measurements(ID)
	errorChan := s.RealtimeClient.Subscribe(ctx, pattern, messages)
	stream := realtime.NewStream(ctx, messages, errorChan, func(msg *realtime.Message) realtime.StreamData[jsonmodels.Measurement] {
		return realtime.StreamData[jsonmodels.Measurement]{
			Action:  msg.Payload.RealtimeAction,
			Channel: msg.Channel,
			Data:    jsonmodels.NewMeasurement(msg.Payload.Data.Bytes()),
		}
	}, func() {
		// Cleanup: unsubscribe from the realtime channel
		s.RealtimeClient.Unsubscribe(pattern)
	})

	return op.OK(stream)
}
