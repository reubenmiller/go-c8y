package operations

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

var ApiOperations = "/devicecontrol/operations"
var ApiOperation = "/devicecontrol/operations/{id}"

var ParamID = "id"

const ResultProperty = "operations"

// Service provides api to get/set/delete operations
type Service struct {
	core.Service
	DeviceResolver *managedobjects.DeviceResolver
}

// NewService creates a new operations service with device resolution capabilities
func NewService(common *core.Service, moService *managedobjects.Service) *Service {
	return &Service{
		Service:        *common,
		DeviceResolver: managedobjects.NewDeviceResolver(moService),
	}
}

// ListOptions to use when search for operations
type ListOptions struct {
	// An agent ID that may be part of the operation. If this parameter is set,
	// the operation response objects contain the deviceExternalIDs object.
	// Use the typed helpers: managedobjects.ByName, ByExternalID, ByQuery, ByID,
	// or cast a string variable with managedobjects.DeviceRef(id).
	AgentID managedobjects.DeviceRef `url:"agentId,omitempty"`

	// The bulk operation ID that this operation belongs to
	BulkOperationID string `url:"bulkOperationId,omitempty"`

	// Start date or date and time of the operation
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the operation
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// The ID of the device the operation is performed for.
	// Use the typed helpers: managedobjects.ByName, ByExternalID, ByQuery, ByID,
	// or cast a string variable with managedobjects.DeviceRef(id).
	DeviceID managedobjects.DeviceRef `url:"deviceId,omitempty"`

	// The type of fragment that must be part of the operation
	FragmentType string `url:"fragmentType,omitempty"`

	// If you are using a range query (that is, at least one of
	// the dateFrom or dateTo parameters is included in the request),
	// then setting revert=true will sort the results by the newest operations
	// first. By default, the results are sorted by the oldest operations first.
	Revert bool `url:"revert,omitempty"`

	// Status of the operation
	Status types.OperationStatus `url:"status,omitempty"`

	pagination.PaginationOptions
}

// OperationIterator provides iteration over operations
type OperationIterator = pagination.Iterator[jsonmodels.Operation]

// List operations
// The DeviceID and AgentID fields support resolver strings:
//   - "12345" - direct ID
//   - "name:deviceName" - lookup by device name
//   - "ext:c8y_Serial:ABC123" - lookup by external ID
//   - "query:type eq 'c8y_Device'" - lookup by inventory query
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Operation] {
	// Resolve DeviceID if it contains a resolver scheme
	if opt.DeviceID != "" && s.DeviceResolver != nil {
		resolutionCtx := ctxhelpers.ResolutionContext(ctx)

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.DeviceID, nil)
		if err != nil {
			return op.Failed[jsonmodels.Operation](err, true)
		}
		opt.DeviceID = managedobjects.DeviceRef(resolvedID)
	}

	// Resolve AgentID if it contains a resolver scheme
	if opt.AgentID != "" && s.DeviceResolver != nil {
		resolutionCtx := ctxhelpers.ResolutionContext(ctx)

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.AgentID, nil)
		if err != nil {
			return op.Failed[jsonmodels.Operation](err, true)
		}
		opt.AgentID = managedobjects.DeviceRef(resolvedID)
	}

	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewOperation)
}

// ListAll returns an iterator for all operations
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *OperationIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Operation] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewOperation,
	)
}

func (s *Service) listB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiOperations)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get an operation
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.Operation] {
	return core.Execute(ctx, s.getB(ID), jsonmodels.NewOperation)
}

func (s *Service) getB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, ID).
		SetURL(ApiOperation)
	return core.NewTryRequest(s.Client, req)
}

// CreateOptions for creating an operation with resolver support
type CreateOptions struct {
	// DeviceID is the target device identifier.
	// Use the typed helpers: managedobjects.ByName, ByExternalID, ByQuery, ByID,
	// or cast a string variable with managedobjects.DeviceRef(id).
	DeviceID managedobjects.DeviceRef

	// Description of the operation
	Description string

	// AdditionalProperties allows for custom fields to be added to the operation
	// Can be a struct, map[string]any, or any JSON-serializable type
	// These properties are deep-merged with the base operation fields
	AdditionalProperties any
}

// Create an operation
// Accepts either CreateOptions (for resolver support and property merging) or any other type (passed through as-is)
//
// Using CreateOptions:
//
//	result := client.Operations.Create(ctx, operations.CreateOptions{
//	    DeviceID: "name:myDevice",  // Resolver string
//	    Description: "Restart device",
//	    AdditionalProperties: map[string]any{
//	        "c8y_Restart": map[string]any{},
//	    },
//	})
//
// Using direct struct/map:
//
//	result := client.Operations.Create(ctx, model.Operation{...})
//	result := client.Operations.Create(ctx, map[string]any{...})
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Operation] {
	// Check if body is CreateOptions - if so, handle resolver and merge logic
	if opts, ok := body.(CreateOptions); ok {
		return s.createWithOptions(ctx, opts)
	}

	// Otherwise, pass through as-is
	return core.Execute(ctx, s.createB(body), jsonmodels.NewOperation)
}

// createWithOptions handles the CreateOptions case with resolver support and property merging
func (s *Service) createWithOptions(ctx context.Context, opts CreateOptions) op.Result[jsonmodels.Operation] {
	// Resolve the device and capture metadata
	deviceID := string(opts.DeviceID)
	meta := make(map[string]any)

	if deviceID != "" && s.DeviceResolver != nil {
		resolutionCtx := ctxhelpers.ResolutionContext(ctx)

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, managedobjects.DeviceRef(deviceID), meta)
		if err != nil {
			return op.Failed[jsonmodels.Operation](err, true)
		}
		deviceID = resolvedID

		// Populate metadata with resolved device information
		meta["id"] = resolvedID
	} else if deviceID != "" {
		// Direct ID provided without resolution
		meta["id"] = deviceID
	}

	// Build base operation from known fields
	baseOperation := map[string]any{
		"deviceId": deviceID,
	}
	if opts.Description != "" {
		baseOperation["description"] = opts.Description
	}

	// Marshal base operation to JSON
	baseJSON, err := json.Marshal(baseOperation)
	if err != nil {
		return op.Failed[jsonmodels.Operation](err, true)
	}

	// If there are additional properties, merge them with the base
	var finalJSON []byte
	if opts.AdditionalProperties != nil {
		additionalJSON, err := json.Marshal(opts.AdditionalProperties)
		if err != nil {
			return op.Failed[jsonmodels.Operation](err, true)
		}

		// Deep merge: additional properties override/extend base properties
		finalJSON, err = jsonUtilities.MergePatch(baseJSON, additionalJSON)
		if err != nil {
			return op.Failed[jsonmodels.Operation](err, true)
		}
	} else {
		finalJSON = baseJSON
	}

	// Create the operation with the merged JSON and add metadata
	result := core.Execute(ctx, s.createBWithJSON(finalJSON), jsonmodels.NewOperation)

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
		SetURL(ApiOperations)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) createBWithJSON(bodyJSON []byte) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(bodyJSON).
		SetURL(ApiOperations)
	return core.NewTryRequest(s.Client, req)
}

// Update an operation
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.Operation] {
	return core.Execute(ctx, s.updateB(ID, body), jsonmodels.NewOperation)
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamID, ID).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiOperation)
	return core.NewTryRequest(s.Client, req)
}

// Delete a list of operations
type DeleteListOptions struct {
	// An agent ID that may be part of the operation.
	// Use the typed helpers: managedobjects.ByName, ByExternalID, ByQuery, ByID,
	// or cast a string variable with managedobjects.DeviceRef(id).
	AgentID managedobjects.DeviceRef `url:"agentId,omitempty"`

	// Start date or date and time of the operation.
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the operation
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// The ID of the device the operation is performed for.
	// Use the typed helpers: managedobjects.ByName, ByExternalID, ByQuery, ByID,
	// or cast a string variable with managedobjects.DeviceRef(id).
	DeviceID managedobjects.DeviceRef `url:"deviceId,omitempty"`

	// Status of the operation
	Status types.OperationStatus `url:"status,omitempty"`
}

// Delete a list of operations
func (s *Service) DeleteList(ctx context.Context, opt DeleteListOptions) op.Result[core.NoContent] {
	// Resolve DeviceID if it contains a resolver scheme
	if opt.DeviceID != "" && s.DeviceResolver != nil {
		resolutionCtx := ctxhelpers.ResolutionContext(ctx)

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.DeviceID, nil)
		if err != nil {
			return op.Failed[core.NoContent](err, true)
		}
		opt.DeviceID = managedobjects.DeviceRef(resolvedID)
	}

	// Resolve AgentID if it contains a resolver scheme
	if opt.AgentID != "" && s.DeviceResolver != nil {
		resolutionCtx := ctxhelpers.ResolutionContext(ctx)

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.AgentID, nil)
		if err != nil {
			return op.Failed[core.NoContent](err, true)
		}
		opt.AgentID = managedobjects.DeviceRef(resolvedID)
	}

	return core.ExecuteNoContent(ctx, s.deleteListB(opt))
}

func (s *Service) deleteListB(opt DeleteListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiOperations)
	return core.NewTryRequest(s.Client, req)
}

// OperationStream provides an iterator for realtime operation subscriptions
type OperationStream = realtime.Stream[realtime.StreamData[jsonmodels.Operation]]

// SubscribeStream subscribes to realtime operations and returns a typed stream iterator.
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
//	streamResult := client.Operations.SubscribeStream(ctx, deviceID)
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
//	    log.Printf("Operation %s: %s - %s", item.Action, item.Data.ID(), item.Data.Status())
//	    if item.Data.Status() == "SUCCESSFUL" {
//	        break
//	    }
//	}
//
//	// Or using range with Seq() (simpler, errors stop iteration)
//	for item := range stream.Seq() {
//	    log.Printf("Operation %s: %s - %s", item.Action, item.Data.ID(), item.Data.Status())
//	}
//	if err := stream.Err(); err != nil {
//	    return err
//	}
func (s *Service) SubscribeStream(ctx context.Context, ID string) op.Result[*OperationStream] {
	err := s.RealtimeClient.Connect()
	if err != nil {
		return op.Failed[*OperationStream](err, false)
	}

	messages := make(chan *realtime.Message, 10)
	pattern := realtime.Operations(ID)
	errorChan := s.RealtimeClient.Subscribe(ctx, pattern, messages)
	stream := realtime.NewStream(ctx, messages, errorChan, func(msg *realtime.Message) realtime.StreamData[jsonmodels.Operation] {
		return realtime.StreamData[jsonmodels.Operation]{
			Action:  msg.Payload.RealtimeAction,
			Channel: msg.Channel,
			Data:    jsonmodels.NewOperation(msg.Payload.Data.Bytes()),
		}
	}, func() {
		// Cleanup: unsubscribe from the realtime channel
		s.RealtimeClient.Unsubscribe(pattern)
	})

	return op.OK(stream)
}
