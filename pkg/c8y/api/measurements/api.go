package measurements

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/source"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/realtime"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

var ApiMeasurements = "/measurement/measurements"
var ApiMeasurement = "/measurement/measurements/{id}"
var ApiMeasurementsSeries = "/measurement/measurements/series"

const ParamID = "id"

const ResultProperty = "measurements"

// Measurement service
type Service struct {
	svc            httpcore.Service
	DeviceResolver *managedobjects.DeviceResolver
}

// NewService creates a new measurement service with device resolution capabilities
func NewService(common *httpcore.Service, moService *managedobjects.Service) *Service {
	return &Service{
		svc:            *common,
		DeviceResolver: managedobjects.NewDeviceResolver(moService),
	}
}

// ListOptions is generated from the OpenAPI spec — see zz_generated_options.go.

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

	return httpcore.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewMeasurement)
}

// ListAll returns an iterator for all measurements. By default it uses the
// time-window keyset optimisation; set ListOptions.Strategy to override.
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *MeasurementIterator {
	// Resolve references once for the whole walk, not once per page.
	if opts.Source != "" && s.DeviceResolver != nil {
		resolvedID, err := s.DeviceResolver.ResolveID(ctxhelpers.ResolutionContext(ctx), opts.Source, nil)
		if err != nil {
			return pagination.NewErrorIterator[jsonmodels.Measurement](err)
		}
		opts.Source = managedobjects.DeviceRef(resolvedID)
	}
	strategy, err := pagination.ResolveTimeStrategy(opts.Strategy)
	if err != nil {
		return pagination.NewErrorIterator[jsonmodels.Measurement](err)
	}
	base := pagination.PageRequest{PaginationOptions: opts.PaginationOptions, Ascending: opts.Revert}
	if opts.Revert {
		base.After = opts.DateFrom
	} else {
		base.Before = opts.DateTo
	}
	return pagination.PaginateWith(
		ctx,
		base,
		strategy,
		func(req pagination.PageRequest) op.Result[jsonmodels.Measurement] {
			o := opts
			o.PaginationOptions = req.PaginationOptions
			o.Revert = req.Ascending
			if req.Ascending {
				if !req.After.IsZero() {
					o.DateFrom = req.After
				}
			} else if !req.Before.IsZero() {
				o.DateTo = req.Before
			}
			return s.List(ctx, o)
		},
		jsonmodels.NewMeasurement,
	)
}

// Export retrieves the measurement collection in an alternative representation
// (e.g. "text/csv" or "application/vnd.ms-excel") selected via the Accept
// header. The server streams the whole matching collection as one document, so
// there is no pagination; the response is returned as a binary payload for the
// caller to stream. The Source resolver field is resolved like List.
func (s *Service) Export(ctx context.Context, opt ListOptions, accept string) op.Result[core.BinaryResponse] {
	if opt.Source != "" && s.DeviceResolver != nil {
		resolutionCtx := ctxhelpers.ResolutionContext(ctx)
		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, nil)
		if err != nil {
			return op.Failed[core.BinaryResponse](err, true)
		}
		opt.Source = managedobjects.DeviceRef(resolvedID)
	}
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", accept).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(ApiMeasurements)
	return httpcore.ExecuteBinary(ctx, httpcore.NewTryRequest(s.svc.Client, req))
}

func (s *Service) listB(opt any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(ApiMeasurements)
	return httpcore.NewTryRequest(s.svc.Client, req, ResultProperty)
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

	// The specific series to delete measurements for.
	ValueFragmentSeries string `url:"valueFragmentSeries,omitempty"`

	// A characteristic which identifies the measurements to delete.
	ValueFragmentType string `url:"valueFragmentType,omitempty"`
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

	return httpcore.ExecuteNoContent(ctx, s.deleteListB(opt))
}

func (s *Service) deleteListB(opt DeleteListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(ApiMeasurements)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Get a measurement by ID
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.Measurement] {
	return httpcore.Execute(ctx, s.getB(ID), jsonmodels.NewMeasurement)
}

func (s *Service) getB(ID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, ID).
		SetURL(ApiMeasurement)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Delete removes a single measurement by ID
func (s *Service) Delete(ctx context.Context, ID string) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.deleteB(ID)).IgnoreNotFound()
}

func (s *Service) deleteB(ID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamID, ID).
		SetURL(ApiMeasurement)
	return httpcore.NewTryRequest(s.svc.Client, req)
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

	// Fragments are custom top-level fields — most importantly the measurement value
	// fragments (e.g. c8y_Temperature). Each is serialised under its FragmentKey() and
	// deep-merged into the body; later entries win. Use a typed model.Fragment, or
	// model.Frag("key", value) for ad-hoc fragments.
	Fragments []model.Fragment
}

// Create posts a new measurement from typed, resolver-aware options.
//
//	result := client.Measurements.Create(ctx, measurements.CreateOptions{
//	    Source: "name:myDevice",  // Resolver string
//	    Type:   "c8y_Temperature",
//	    Time:   time.Now(),
//	    Fragments: []model.Fragment{
//	        model.Frag("c8y_Temperature", map[string]any{
//	            "T": map[string]any{"value": 23.5, "unit": "°C"},
//	        }),
//	    },
//	})
//
// To send a fully pre-built body, use CreateRaw.
func (s *Service) Create(ctx context.Context, opts CreateOptions) op.Result[jsonmodels.Measurement] {
	return s.createWithOptions(ctx, opts)
}

// CreateRaw posts a measurement from a pre-built body, passed through to the API as-is,
// except that a device reference at "source.id" that is not an id (e.g. "dev01",
// "name:dev01", "ext:c8y_Serial:ABC") is resolved to the device id first.
//
//	result := client.Measurements.CreateRaw(ctx, map[string]any{...})
func (s *Service) CreateRaw(ctx context.Context, body any) op.Result[jsonmodels.Measurement] {
	body, err := s.DeviceResolver.ResolveBodyRefs(ctx, body, "source.id")
	if err != nil {
		return op.Failed[jsonmodels.Measurement](err, false)
	}
	return httpcore.Execute(ctx, s.createB(body), jsonmodels.NewMeasurement)
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
	baseMeasurement := map[string]any{
		"source": map[string]any{"id": sourceID},
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

	// Deep-merge each custom fragment under its key; later entries win.
	finalJSON, err := model.MergeFragments(baseJSON, opts.Fragments)
	if err != nil {
		return op.Failed[jsonmodels.Measurement](err, true)
	}

	// Create the measurement with the merged JSON and add metadata
	result := httpcore.Execute(ctx, s.createBWithJSON(finalJSON), jsonmodels.NewMeasurement)

	// Add resolver metadata to result
	if result.Meta == nil {
		result.Meta = make(map[string]any)
	}
	for k, v := range meta {
		result.Meta[k] = v
	}

	return result
}

func (s *Service) createB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiMeasurements)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

func (s *Service) createBWithJSON(bodyJSON []byte) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(bodyJSON).
		SetURL(ApiMeasurements)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// CreateList creates multiple measurements in one request. The body is a
// measurement collection, {"measurements": [...]}. As in CreateRaw, a device
// reference at "source.id" of a measurement that is not an id is resolved to
// the device id first (each reference once).
func (s *Service) CreateList(ctx context.Context, body any) op.Result[jsonmodels.Measurement] {
	body, err := s.resolveListSources(ctx, body)
	if err != nil {
		return op.Failed[jsonmodels.Measurement](err, false)
	}
	return httpcore.ExecuteCollection(ctx, s.createListB(body), ResultProperty, "", jsonmodels.NewMeasurement)
}

// resolveListSources resolves the device references of the measurements in a
// measurement collection.
func (s *Service) resolveListSources(ctx context.Context, body any) (any, error) {
	if s.DeviceResolver == nil || body == nil {
		return body, nil
	}
	var raw []byte
	switch b := body.(type) {
	case json.RawMessage:
		raw = b
	case []byte:
		raw = b
	case string:
		raw = []byte(b)
	default:
		var err error
		if raw, err = json.Marshal(body); err != nil {
			return body, err
		}
	}
	n := gjson.GetBytes(raw, ResultProperty+".#").Int()
	paths := make([]string, n)
	for i := range paths {
		paths[i] = fmt.Sprintf("%s.%d.source.id", ResultProperty, i)
	}
	ids := map[string]string{}
	return source.ResolveBodyRefs(ctx, body, func(ctx context.Context, ref string) (string, error) {
		if id, ok := ids[ref]; ok {
			return id, nil
		}
		id, err := s.DeviceResolver.ResolveID(ctxhelpers.ResolutionContext(ctx), managedobjects.DeviceRef(ref), nil)
		if err == nil {
			ids[ref] = id
		}
		return id, err
	}, paths...)
}

func (s *Service) createListB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetContentType(types.MimeTypeCumulocityMeasurementCollection).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiMeasurements)
	return httpcore.NewTryRequest(s.svc.Client, req, ResultProperty)
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

	AggregationType model.MeasurementAggregationType `url:"aggregationType,omitempty"`

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
	result := httpcore.Execute(ctx, s.listSeriesB(opt), jsonmodels.NewMeasurementSeries)
	if result.Err == nil {
		// Populate device info in the measurement series
		result.Data.DeviceID = deviceID
		result.Data.DeviceName = deviceName
	}
	return result
}

func (s *Service) listSeriesB(opt ListSeriesOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(ApiMeasurementsSeries)
	return httpcore.NewTryRequest(s.svc.Client, req)
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
	// A device reference that is not an id (or "*") is resolved first.
	if ID != "" && ID != "*" && !source.IsID(ID) {
		resolvedID, err := s.DeviceResolver.ResolveID(ctxhelpers.ResolutionContext(ctx), managedobjects.DeviceRef(source.ParseRef(ID)), nil)
		if err != nil {
			return op.Failed[*MeasurementStream](err, false)
		}
		ID = resolvedID
	}
	pattern := realtime.Measurements(ID)
	err := s.svc.RealtimeClient.ConnectContext(ctx)
	if err != nil {
		if dry, ok := realtime.DryRunStream[realtime.StreamData[jsonmodels.Measurement]](ctx, err, pattern); ok {
			return dry
		}
		return op.Failed[*MeasurementStream](err, false)
	}

	messages := make(chan *realtime.Message, 10)
	errorChan := s.svc.RealtimeClient.Subscribe(ctx, pattern, messages)
	stream := realtime.NewStream(ctx, messages, errorChan, func(msg *realtime.Message) realtime.StreamData[jsonmodels.Measurement] {
		return realtime.StreamData[jsonmodels.Measurement]{
			Action:  msg.Payload.RealtimeAction,
			Channel: msg.Channel,
			Data:    jsonmodels.NewMeasurement(msg.Payload.Data.Bytes()),
		}
	}, func() {
		// Cleanup: unsubscribe from the realtime channel
		s.svc.RealtimeClient.Unsubscribe(pattern)
	})

	return op.OK(stream)
}
