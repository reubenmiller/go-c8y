package alarms

import (
	"context"
	"encoding/json"
	"time"

	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/api/contexthelpers"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/realtime"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/pkg/jsonUtilities"
	"resty.dev/v3"
)

var ApiAlarms = "/alarm/alarms"
var ApiAlarmsCount = "/alarm/alarms/count"
var ApiAlarm = "/alarm/alarms/{id}"

var ParamId = "id"

const ResultProperty = "alarms"

// Service provides api to get/set/delete audit entries in Cumulocity
type Service struct {
	core.Service
	DeviceResolver *managedobjects.DeviceResolver
}

// NewService creates a new alarms service with device resolution capabilities
func NewService(common *core.Service, moService *managedobjects.Service) *Service {
	return &Service{
		Service:        *common,
		DeviceResolver: managedobjects.NewDeviceResolver(moService),
	}
}

// ListOptions to use when search for alarms
type ListOptions struct {
	// Start date or date and time of the alarm creation
	CreatedFrom time.Time `url:"createdFrom,omitempty,omitzero"`

	// End date or date and time of the alarm creation
	CreatedTo time.Time `url:"createdTo,omitempty,omitzero"`

	// Start date or date and time of the last update made
	LastUpdatedFrom time.Time `url:"lastUpdatedFrom,omitempty,omitzero"`

	// End date or date and time of the last update made
	LastUpdatedTo time.Time `url:"lastUpdatedTo,omitempty,omitzero"`

	// Start date or date and time of the alarm occurrence
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the alarm occurrence
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// Source device to filter measurements by.
	// Supports resolver strings: direct ID, "name:deviceName", "ext:type:id", "query:..."
	Source string `url:"source,omitempty"`

	// The types of alarm to search for
	Type []string `url:"type,omitempty"`

	// The status of the alarm to search for. Should not be used when resolved parameter is provided
	Status []model.AlarmStatus `url:"status,omitempty"`

	// The severity of the alarm to search for
	Severity []model.AlarmSeverity `url:"severity,omitempty"`

	// When set to true only alarms with status CLEARED will be fetched, whereas false will fetch all
	// alarms with status ACTIVE or ACKNOWLEDGED. Takes precedence over the status parameter
	Resolved bool `url:"resolved,omitempty"`

	// When set to true, alarms for related source assets, devices and additions will
	// also be included in the response. When this parameter is provided a source
	// must be specified.
	WithSourceChildren bool `url:"withSourceChildren,omitempty"`

	// When set to true, alarms for related source assets will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAssets bool `url:"withSourceAssets,omitempty"`

	// When set to true, alarms for related source devices will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceDevices bool `url:"withSourceDevices,omitempty"`

	// When set to true, alarms for related source additions will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAdditions bool `url:"withSourceAdditions,omitempty"`

	pagination.PaginationOptions
}

// AlarmIterator provides iteration over alarms
type AlarmIterator = pagination.Iterator[jsonmodels.Alarm]

// List alarms
// The Source field supports resolver strings:
//   - "12345" - direct ID
//   - "name:deviceName" - lookup by device name
//   - "ext:c8y_Serial:ABC123" - lookup by external ID
//   - "query:type eq 'c8y_Device'" - lookup by inventory query
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Alarm] {
	// Resolve Source if it contains a resolver scheme
	if opt.Source != "" && s.DeviceResolver != nil {
		resolutionCtx := ctxhelpers.ResolutionContext(ctx)

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, nil)
		if err != nil {
			return op.Failed[jsonmodels.Alarm](err, true)
		}
		opt.Source = resolvedID
	}

	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewAlarm)
}

// ListAll returns an iterator for all alarms
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *AlarmIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Alarm] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewAlarm,
	)
}

func (s *Service) listB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiAlarms)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// CountOptions to use when counting the active alarms
type CountOptions struct {
	// Start date or date and time of the alarm occurrence
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the alarm occurrence
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// When set to true only alarms with status CLEARED will be fetched, whereas false will fetch all
	// alarms with status ACTIVE or ACKNOWLEDGED. Takes precedence over the status parameter
	Resolved bool `url:"resolved,omitempty"`

	// The severity of the alarm to search for
	Severity []string `url:"severity,omitempty"`

	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// The status of the alarm to search for. Should not be used when resolved parameter is provided
	Status []string `url:"status,omitempty"`

	// The types of alarm to search for
	Type []string `url:"type,omitempty"`
}

// Get an alarm
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.Alarm] {
	return core.Execute(ctx, s.getB(ID), jsonmodels.NewAlarm)
}

func (s *Service) getB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, ID).
		SetURL(ApiAlarm)
	return core.NewTryRequest(s.Client, req)
}

// CreateOptions for creating an alarm with resolver support
type CreateOptions struct {
	// Source device identifier (supports resolver strings)
	// Examples: "12345", "name:deviceName", "ext:c8y_Serial:ABC", "query:..."
	Source string

	// Type of the alarm
	Type string

	// Text description of the alarm
	Text string

	// Severity of the alarm (CRITICAL, MAJOR, MINOR, WARNING)
	Severity string

	// Status of the alarm (ACTIVE, ACKNOWLEDGED, CLEARED)
	Status string

	// Time when the alarm occurred
	Time time.Time

	// AdditionalProperties allows for custom fields to be added to the alarm
	// Can be a struct, map[string]interface{}, or any JSON-serializable type
	// These properties are deep-merged with the base alarm fields
	AdditionalProperties interface{}
}

// Create an alarm
// Accepts either CreateOptions (for resolver support and property merging) or any other type (passed through as-is)
//
// Using CreateOptions:
//
//	result := client.Alarms.Create(ctx, alarms.CreateOptions{
//	    Source: "name:myDevice",  // Resolver string
//	    Type: "c8y_TestAlarm",
//	    Text: "Test alarm",
//	    AdditionalProperties: map[string]interface{}{"custom": "value"},
//	})
//
// Using direct struct/map:
//
//	result := client.Alarms.Create(ctx, model.Alarm{...})
//	result := client.Alarms.Create(ctx, map[string]interface{}{...})
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Alarm] {
	// Check if body is CreateOptions - if so, handle resolver and merge logic
	if opts, ok := body.(CreateOptions); ok {
		return s.createWithOptions(ctx, opts)
	}

	// Otherwise, pass through as-is
	return core.Execute(ctx, s.createB(body), jsonmodels.NewAlarm)
}

// createWithOptions handles the CreateOptions case with resolver support and property merging
func (s *Service) createWithOptions(ctx context.Context, opts CreateOptions) op.Result[jsonmodels.Alarm] {
	// Resolve the source device and capture metadata
	sourceID := opts.Source
	meta := make(map[string]any)

	if sourceID != "" && s.DeviceResolver != nil {
		resolutionCtx := ctxhelpers.ResolutionContext(ctx)

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, sourceID, meta)
		if err != nil {
			return op.Failed[jsonmodels.Alarm](err, true)
		}
		sourceID = resolvedID

		// Populate metadata with resolved device information
		meta["id"] = resolvedID
		if name, ok := meta["name"].(string); ok {
			// name is already in meta from ResolveID
			_ = name
		}
	} else if sourceID != "" {
		// Direct ID provided without resolution
		meta["id"] = sourceID
	}

	// Build base alarm from known fields
	baseAlarm := map[string]interface{}{
		"source": map[string]interface{}{"id": sourceID},
	}
	if opts.Type != "" {
		baseAlarm["type"] = opts.Type
	}
	if opts.Text != "" {
		baseAlarm["text"] = opts.Text
	}
	if opts.Severity != "" {
		baseAlarm["severity"] = opts.Severity
	}
	if opts.Status != "" {
		baseAlarm["status"] = opts.Status
	}
	if opts.Time.IsZero() {
		baseAlarm["time"] = time.Now()
	} else {
		baseAlarm["time"] = opts.Time
	}

	// Marshal base alarm to JSON
	baseJSON, err := json.Marshal(baseAlarm)
	if err != nil {
		return op.Failed[jsonmodels.Alarm](err, true)
	}

	// If there are additional properties, merge them with the base
	var finalJSON []byte
	if opts.AdditionalProperties != nil {
		additionalJSON, err := json.Marshal(opts.AdditionalProperties)
		if err != nil {
			return op.Failed[jsonmodels.Alarm](err, true)
		}

		// Deep merge: additional properties override/extend base properties
		finalJSON, err = jsonUtilities.MergePatch(baseJSON, additionalJSON)
		if err != nil {
			return op.Failed[jsonmodels.Alarm](err, true)
		}
	} else {
		finalJSON = baseJSON
	}

	// Create the alarm with the merged JSON and add metadata
	result := core.Execute(ctx, s.createBWithJSON(finalJSON), jsonmodels.NewAlarm)

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
		SetContentType(types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiAlarms)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) createBWithJSON(bodyJSON []byte) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetBody(bodyJSON).
		SetURL(ApiAlarms)
	return core.NewTryRequest(s.Client, req)
}

// Update an alarm
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.Alarm] {
	return core.Execute(ctx, s.updateB(ID, body), jsonmodels.NewAlarm)
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiAlarm)
	return core.NewTryRequest(s.Client, req)
}

// UpdateOptions option when bulk updating alarms
type BulkUpdateOptions struct {
	// Start date or date and time of the alarm creation
	CreatedFrom time.Time `url:"createdFrom,omitempty,omitzero"`

	// End date or date and time of the alarm creation
	CreatedTo time.Time `url:"createdTo,omitempty,omitzero"`

	// Start date or date and time of the alarm occurrence
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the alarm occurrence
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// When set to true only alarms with status CLEARED will be fetched, whereas false will fetch all
	// alarms with status ACTIVE or ACKNOWLEDGED. Takes precedence over the status parameter
	Resolved bool `url:"resolved,omitempty"`

	// The severity of the alarm to search for
	Severity []string `url:"severity,omitempty"`

	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// TODO: Check if this is supported or not
	// The types of alarm to search for
	// Type []string `url:"type,omitempty"`

	// The status of the alarm to search for. Should not be used when resolved parameter is provided
	Status []string `url:"status,omitempty"`

	// When set to true, alarms for related source assets, devices and additions will
	// also be included in the response. When this parameter is provided a source
	// must be specified.
	WithSourceChildren bool `url:"withSourceChildren,omitempty"`

	// When set to true, alarms for related source assets will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAssets bool `url:"withSourceAssets,omitempty"`

	// When set to true, alarms for related source devices will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceDevices bool `url:"withSourceDevices,omitempty"`

	// When set to true, alarms for related source additions will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAdditions bool `url:"withSourceAdditions,omitempty"`
}

// BulkUpdateAlarms bulk update of alarm collection
// The PUT method allows for updating alarms collections. Currently only the status of alarms can be changed.
// Response status:
// 200 - if the process has completed, all alarms have been updated
// 202 - if process continues in background
//
// Since this operations can take a lot of time, request returns after maximum 0.5 sec of processing, and updating is continued as a background process in the platform.
func (s *Service) UpdateList(ctx context.Context, opt BulkUpdateOptions, body any) op.Result[jsonmodels.Alarm] {
	// Resolve Source if it contains a resolver scheme
	if opt.Source != "" && s.DeviceResolver != nil {
		resolutionCtx := ctxhelpers.ResolutionContext(ctx)

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, nil)
		if err != nil {
			return op.Failed[jsonmodels.Alarm](err, true)
		}
		opt.Source = resolvedID
	}

	return core.Execute(ctx, s.updateListB(opt, body), jsonmodels.NewAlarm)
}

func (s *Service) updateListB(opt BulkUpdateOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiAlarms)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// DeleteListOptions option when deleting a collection of alarms
type DeleteListOptions struct {
	// Start date or date and time of the alarm creation
	CreatedFrom time.Time `url:"createdFrom,omitempty,omitzero"`

	// End date or date and time of the alarm creation
	CreatedTo time.Time `url:"createdTo,omitempty,omitzero"`

	// Start date or date and time of the alarm occurrence
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the alarm occurrence
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// When set to true only alarms with status CLEARED will be fetched, whereas false will fetch all
	// alarms with status ACTIVE or ACKNOWLEDGED. Takes precedence over the status parameter
	Resolved bool `url:"resolved,omitempty"`

	// The severity of the alarm to search for
	Severity []string `url:"severity,omitempty"`

	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// TODO: Check if this is supported or not
	// The types of alarm to search for
	// Type []string `url:"type,omitempty"`

	// The status of the alarm to search for. Should not be used when resolved parameter is provided
	Status []string `url:"status,omitempty"`

	// The types of alarm to search for
	Type []string `url:"type,omitempty"`

	// When set to true, alarms for related source assets, devices and additions will
	// also be included in the response. When this parameter is provided a source
	// must be specified.
	WithSourceChildren bool `url:"withSourceChildren,omitempty"`

	// When set to true, alarms for related source assets will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAssets bool `url:"withSourceAssets,omitempty"`

	// When set to true, alarms for related source devices will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceDevices bool `url:"withSourceDevices,omitempty"`

	// When set to true, alarms for related source additions will also be included in
	// the response. When this parameter is provided a source must be specified.
	WithSourceAdditions bool `url:"withSourceAdditions,omitempty"`
}

// Remove alarm collections specified by query parameters
func (s *Service) DeleteList(ctx context.Context, opt DeleteListOptions) op.Result[core.NoContent] {
	// Resolve Source if it contains a resolver scheme
	if opt.Source != "" && s.DeviceResolver != nil {
		resolutionCtx := ctxhelpers.ResolutionContext(ctx)

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
		SetURL(ApiAlarms)
	return core.NewTryRequest(s.Client, req)
}

// Subscribe subscribes to realtime alarms and sends raw messages to the provided channel.
// The subscription automatically unsubscribes when the context is cancelled or times out.
func (s *Service) Subscribe(ctx context.Context, ID string, out chan<- *realtime.Message) op.Result[chan error] {
	err := s.RealtimeClient.Connect()
	if err != nil {
		return op.Failed[chan error](err, false)
	}
	return op.OK(s.RealtimeClient.Subscribe(ctx, realtime.Alarms(ID), out))
}

// AlarmStream provides an iterator for realtime alarm subscriptions
type AlarmStream = realtime.Stream[realtime.StreamData[jsonmodels.Alarm]]

// SubscribeStream subscribes to realtime alarms and returns a typed stream iterator.
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
//	streamResult := client.Alarms.SubscribeStream(ctx, deviceID)
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
//	    log.Printf("Alarm %s: %s (severity: %s)", item.Action, item.Data.Type(), item.Data.Severity())
//	    if item.Data.Severity() == "CRITICAL" {
//	        break
//	    }
//	}
//
//	// Or using range with Seq() (simpler, errors stop iteration)
//	for item := range stream.Seq() {
//	    log.Printf("Alarm %s: %s", item.Action, item.Data.Type())
//	}
//	if err := stream.Err(); err != nil {
//	    return err
//	}
func (s *Service) SubscribeStream(ctx context.Context, ID string) op.Result[*AlarmStream] {
	err := s.RealtimeClient.Connect()
	if err != nil {
		return op.Failed[*AlarmStream](err, false)
	}

	messages := make(chan *realtime.Message, 10)
	pattern := realtime.Alarms(ID)
	errorChan := s.RealtimeClient.Subscribe(ctx, pattern, messages)
	stream := realtime.NewStream(ctx, messages, errorChan, func(msg *realtime.Message) realtime.StreamData[jsonmodels.Alarm] {
		return realtime.StreamData[jsonmodels.Alarm]{
			Action:  msg.Payload.RealtimeAction,
			Channel: msg.Channel,
			Data:    jsonmodels.NewAlarm(msg.Payload.Data.Bytes()),
		}
	}, func() {
		// Cleanup: unsubscribe from the realtime channel
		s.RealtimeClient.Unsubscribe(pattern)
	})

	return op.OK(stream)
}
