package events

import (
	"context"
	"encoding/json"
	"time"

	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/api/contexthelpers"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/events/eventbinaries"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/realtime"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/pkg/jsonUtilities"
	"resty.dev/v3"
)

var ApiEvents = "/event/events"
var ApiEvent = "/event/events/{id}"

var ParamId = "id"

const ResultProperty = "events"

func NewService(s *core.Service, moService *managedobjects.Service) *Service {
	return &Service{
		Service:        *s,
		Binaries:       eventbinaries.NewService(s),
		DeviceResolver: managedobjects.NewDeviceResolver(moService),
	}
}

// Service provides api to get/set/delete events in Cumulocity
type Service struct {
	core.Service
	Binaries       *eventbinaries.Service
	DeviceResolver *managedobjects.DeviceResolver
}

// ListOptions to use when search for events
type ListOptions struct {
	// Start date or date and time of the event's creation (set by the platform during creation)
	CreatedFrom time.Time `url:"createdFrom,omitempty,omitzero"`

	// End date or date and time of the event's creation (set by the platform during creation)
	CreatedTo time.Time `url:"createdTo,omitempty,omitzero"`

	// Start date or date and time of the last update made
	LastUpdatedFrom time.Time `url:"lastUpdatedFrom,omitempty,omitzero"`

	// End date or date and time of the last update made
	LastUpdatedTo time.Time `url:"lastUpdatedTo,omitempty,omitzero"`

	// Start date or date and time of the event occurrence (provided by the device).
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the last update made
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// Allows filtering events by the fragment's value, but only
	// when provided together with fragmentType.
	FragmentType string `url:"fragmentType,omitempty"`

	// Allows filtering events by the fragment's value, but only
	// when provided together with fragmentType.
	// Important: Only fragments with a string value are supported.
	FragmentValue string `url:"fragmentValue,omitempty"`

	// If you are using a range query (that is, at least one of
	// the dateFrom or dateTo parameters is included in the request),
	// then setting revert=true will sort the results by the oldest
	// events first. By default, the results are sorted by the newest
	// events first.
	Revert bool `url:"revert,omitempty"`

	// The managed object ID to which the event is associated.
	// Supports resolver strings: direct ID, "name:deviceName", "ext:type:id", "query:..."
	Source string `url:"source,omitempty"`

	// The type of event to search for
	Type string `url:"type,omitempty"`

	// When set to true, events for related source assets, devices and additions will
	// also be included in the response. When this parameter is provided a source
	// must be specified.
	WithSourceChildren bool `url:"withSourceChildren,omitempty"`

	// When set to true, events for related source assets will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAssets bool `url:"withSourceAssets,omitempty"`

	// When set to true, events for related source devices will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceDevices bool `url:"withSourceDevices,omitempty"`

	// When set to true, events for related source additions will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAdditions bool `url:"withSourceAdditions,omitempty"`

	pagination.PaginationOptions
}

// EventIterator provides iteration over events
type EventIterator = pagination.Iterator[jsonmodels.Event]

// List events
// The Source field supports resolver strings:
//   - "12345" - direct ID
//   - "name:deviceName" - lookup by device name
//   - "ext:c8y_Serial:ABC123" - lookup by external ID
//   - "query:type eq 'c8y_Device'" - lookup by inventory query
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Event] {
	// Resolve Source if it contains a resolver scheme
	if opt.Source != "" && s.DeviceResolver != nil {
		resolutionCtx := ctx
		if ctxhelpers.IsDeferredExecution(ctx) {
			resolutionCtx = context.Background()
		}

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, nil)
		if err != nil {
			return op.Failed[jsonmodels.Event](err, true)
		}
		opt.Source = resolvedID
	}

	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewEvent)
}

// ListAll returns an iterator for all events
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *EventIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Event] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewEvent,
	)
}

func (s *Service) listB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiEvents)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get an event
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.Event] {
	return core.Execute(ctx, s.getB(ID), jsonmodels.NewEvent)
}

func (s *Service) getB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiEvent)
	return core.NewTryRequest(s.Client, req)
}

// CreateOptions for creating an event with resolver support
type CreateOptions struct {
	// Source device identifier (supports resolver strings)
	// Examples: "12345", "name:deviceName", "ext:c8y_Serial:ABC", "query:..."
	Source string

	// Type of the event
	Type string

	// Text description of the event
	Text string

	// Time when the event occurred
	Time time.Time

	// AdditionalProperties allows for custom fields to be added to the event
	// Can be a struct, map[string]interface{}, or any JSON-serializable type
	// These properties are deep-merged with the base event fields
	AdditionalProperties interface{}
}

// Create an event
// Accepts either CreateOptions (for resolver support and property merging) or any other type (passed through as-is)
//
// Using CreateOptions:
//
//	result := client.Events.Create(ctx, events.CreateOptions{
//	    Source: "name:myDevice",  // Resolver string
//	    Type: "c8y_TestEvent",
//	    Text: "Test event",
//	    AdditionalProperties: map[string]interface{}{"custom": "value"},
//	})
//
// Using direct struct/map:
//
//	result := client.Events.Create(ctx, model.Event{...})
//	result := client.Events.Create(ctx, map[string]interface{}{...})
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Event] {
	// Check if body is CreateOptions - if so, handle resolver and merge logic
	if opts, ok := body.(CreateOptions); ok {
		return s.createWithOptions(ctx, opts)
	}

	// Otherwise, pass through as-is
	return core.Execute(ctx, s.createB(body), jsonmodels.NewEvent)
}

// createWithOptions handles the CreateOptions case with resolver support and property merging
func (s *Service) createWithOptions(ctx context.Context, opts CreateOptions) op.Result[jsonmodels.Event] {
	// Resolve the source device and capture metadata
	sourceID := opts.Source
	meta := make(map[string]any)

	if sourceID != "" && s.DeviceResolver != nil {
		resolutionCtx := ctx
		if ctxhelpers.IsDeferredExecution(ctx) {
			// Create a new context that preserves mock responses but not deferred execution
			resolutionCtx = ctxhelpers.WithMockResponses(context.Background(), ctxhelpers.IsMockResponses(ctx))
		}

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, sourceID, meta)
		if err != nil {
			return op.Failed[jsonmodels.Event](err, true)
		}
		sourceID = resolvedID

		// Populate metadata with resolved device information
		meta["id"] = resolvedID
	} else if sourceID != "" {
		// Direct ID provided without resolution
		meta["id"] = sourceID
	}

	// Build base event from known fields
	baseEvent := map[string]interface{}{
		"source": map[string]interface{}{"id": sourceID},
	}
	if opts.Type != "" {
		baseEvent["type"] = opts.Type
	}
	if opts.Text != "" {
		baseEvent["text"] = opts.Text
	}
	if !opts.Time.IsZero() {
		baseEvent["time"] = opts.Time
	}

	// Marshal base event to JSON
	baseJSON, err := json.Marshal(baseEvent)
	if err != nil {
		return op.Failed[jsonmodels.Event](err, true)
	}

	// If there are additional properties, merge them with the base
	var finalJSON []byte
	if opts.AdditionalProperties != nil {
		additionalJSON, err := json.Marshal(opts.AdditionalProperties)
		if err != nil {
			return op.Failed[jsonmodels.Event](err, true)
		}

		// Deep merge: additional properties override/extend base properties
		finalJSON, err = jsonUtilities.MergePatch(baseJSON, additionalJSON)
		if err != nil {
			return op.Failed[jsonmodels.Event](err, true)
		}
	} else {
		finalJSON = baseJSON
	}

	// Create the event with the merged JSON and add metadata
	result := core.Execute(ctx, s.createBWithJSON(finalJSON), jsonmodels.NewEvent)

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
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiEvents)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) createBWithJSON(bodyJSON []byte) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(bodyJSON).
		SetURL(ApiEvents)
	return core.NewTryRequest(s.Client, req)
}

// Update an event
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.Event] {
	return core.Execute(ctx, s.updateB(ID, body), jsonmodels.NewEvent)
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiEvent)
	return core.NewTryRequest(s.Client, req)
}

// Delete removes an event by its ID
func (s *Service) Delete(ctx context.Context, ID string) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(ID))
}

func (s *Service) deleteB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, ID).
		SetURL(ApiEvent)
	return core.NewTryRequest(s.Client, req)
}

// DeleteListOptions option when deleting a collection of events
type DeleteListOptions struct {
	// Start date or date and time of the event's creation (set by the platform during creation)
	CreatedFrom time.Time `url:"createdFrom,omitempty,omitzero"`

	// End date or date and time of the event's creation (set by the platform during creation)
	CreatedTo time.Time `url:"createdTo,omitempty,omitzero"`

	// Start date or date and time of the event occurrence (provided by the device).
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the last update made
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// Allows filtering events by the fragment's value, but only
	// when provided together with fragmentType.
	FragmentType string `url:"fragmentType,omitempty"`

	// The managed object ID to which the event is associated
	Source string `url:"source,omitempty"`

	// The type of event to search for
	Type string `url:"type,omitempty"`
}

// Remove event collections specified by query parameters
//
// DELETE requests are not synchronous. The response could be returned
// before the delete request has been completed. This may happen especially
// when the deleted event has a lot of associated data. After sending the
// request, the platform starts deleting the associated data in an asynchronous way.
// Finally, the requested event is deleted after all associated data has been deleted.
func (s *Service) DeleteList(ctx context.Context, opt DeleteListOptions) op.Result[core.NoContent] {
	// Resolve Source if it contains a resolver scheme
	if opt.Source != "" && s.DeviceResolver != nil {
		resolutionCtx := ctx
		if ctxhelpers.IsDeferredExecution(ctx) {
			resolutionCtx = context.Background()
		}

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, nil)
		if err != nil {
			return op.Failed[core.NoContent](err, true)
		}
		opt.Source = resolvedID
	}

	return core.ExecuteNoContent(ctx, s.deleteListB(opt))
}

func (s *Service) deleteListB(opt DeleteListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiEvents)
	return core.NewTryRequest(s.Client, req)
}

// EventStream provides an iterator for realtime event subscriptions
type EventStream = realtime.Stream[realtime.StreamData[jsonmodels.Event]]

// SubscribeStream subscribes to realtime events and returns a typed stream iterator.
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
//	streamResult := client.Events.SubscribeStream(ctx, deviceID)
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
//	    log.Printf("Event %s: %s", item.Action, item.Data.Type())
//	    if item.Data.Type() == "targetType" {
//	        break
//	    }
//	}
//
//	// Or using range with Seq() (simpler, errors stop iteration)
//	for item := range stream.Seq() {
//	    log.Printf("Event %s: %s", item.Action, item.Data.Type())
//	}
//	if err := stream.Err(); err != nil {
//	    return err
//	}
func (s *Service) SubscribeStream(ctx context.Context, ID string) op.Result[*EventStream] {
	err := s.RealtimeClient.Connect()
	if err != nil {
		return op.Failed[*EventStream](err, false)
	}

	messages := make(chan *realtime.Message, 10)
	pattern := realtime.Events(ID)
	errorChan := s.RealtimeClient.Subscribe(ctx, pattern, messages)
	stream := realtime.NewStream(ctx, messages, errorChan, func(msg *realtime.Message) realtime.StreamData[jsonmodels.Event] {
		return realtime.StreamData[jsonmodels.Event]{
			Action:  msg.Payload.RealtimeAction,
			Channel: msg.Channel,
			Data:    jsonmodels.NewEvent(msg.Payload.Data.Bytes()),
		}
	}, func() {
		// Cleanup: unsubscribe from the realtime channel
		s.RealtimeClient.Unsubscribe(pattern)
	})

	return op.OK(stream)
}
