package managedobjects

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/binaries"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/identity"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects/childadditions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects/childassets"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects/childdevices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/realtime"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/source"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiManagedObjects = "/inventory/managedObjects"
var ApiManagedObject = "/inventory/managedObjects/{id}"
var ApiManagedObjectSupportedMeasurements = "/inventory/managedObjects/{id}/supportedMeasurements"
var ApiManagedObjectSupportedSeries = "/inventory/managedObjects/{id}/supportedSeries"

const ParamId = "id"

const ResultProperty = "managedObjects"

func NewService(s *core.Service) *Service {
	service := &Service{
		Service:         *s,
		identityService: identity.NewService(s),
		binariesService: binaries.NewService(s),
		ChildAdditions:  childadditions.NewService(s),
		ChildAssets:     childassets.NewService(s),
		ChildDevices:    childdevices.NewService(s),
		customResolvers: make(map[string]source.Resolver),
	}

	// Setup lookup functions for resolvers
	service.lookupByExternalID = func(ctx context.Context, typ, extID string) (string, map[string]any, error) {
		result := service.identityService.Get(ctx, identity.IdentityOptions{
			Type:       typ,
			ExternalID: extID,
		})
		if result.Err != nil {
			return "", nil, result.Err
		}
		return result.Data.ManagedObjectID(), map[string]any{
			"externalType": typ,
			"externalID":   extID,
		}, nil
	}

	service.lookupByName = func(ctx context.Context, name string) (string, map[string]any, error) {
		opts := ListOptions{}
		opts.PaginationOptions.PageSize = 1
		opts.Query = model.NewInventoryQuery().
			AddFilterEqStr("name", name).
			AddOrderBy("name").
			AddOrderBy("creationTime").
			Build()

		result := service.List(ctx, opts)
		if result.Err != nil {
			return "", nil, result.Err
		}

		for item := range op.Iter(result) {
			return item.ID(), map[string]any{
				"id":    item.ID(),
				"name":  item.Name(),
				"owner": item.Owner(),
			}, nil
		}

		return "", nil, fmt.Errorf("managed object not found with name: %s", name)
	}

	service.lookupByQuery = func(ctx context.Context, query string) (string, map[string]any, error) {
		opts := ListOptions{}
		opts.PaginationOptions.PageSize = 1
		opts.Query = query

		result := service.List(ctx, opts)
		if result.Err != nil {
			return "", nil, result.Err
		}

		for item := range result.Data.Iter() {
			obj := jsonmodels.NewManagedObject(item.Bytes())
			return obj.ID(), map[string]any{"query": query}, nil
		}

		return "", nil, fmt.Errorf("managed object not found with query: %s", query)
	}

	return service
}

// Service inventory api to interact with managed objects
type Service struct {
	core.Service
	identityService *identity.Service
	binariesService *binaries.Service
	ChildAdditions  *childadditions.Service
	ChildAssets     *childassets.Service
	ChildDevices    *childdevices.Service

	// Resolver lookup functions
	lookupByExternalID func(ctx context.Context, typ, extID string) (string, map[string]any, error)
	lookupByName       func(ctx context.Context, name string) (string, map[string]any, error)
	lookupByQuery      func(ctx context.Context, query string) (string, map[string]any, error)
	customResolvers    map[string]source.Resolver
}

// ListOptions filter managed object
type ListOptions struct {
	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	Text string `url:"text,omitempty"`

	// Read-only collection of managed objects fetched for a given list of ids (placeholder {ids}),for example "?ids=41,43,68".
	Ids []string `url:"ids,omitempty"`

	Query string `url:"query,omitempty"`

	DeviceQuery string `url:"q,omitempty"`

	GetOptions

	// Pagination options
	pagination.PaginationOptions
}

// List managed objects
func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiManagedObjects)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type GetOptions struct {
	WithParents       bool `url:"withParents,omitempty"`
	WithChildren      bool `url:"withChildren,omitempty"`
	withChildrenCount bool `url:"withChildrenCount,omitempty"`
	SkipChildrenNames bool `url:"skipChildrenNames,omitempty"`
	WithLatestValues  bool `url:"withLatestValues,omitempty"`
}

// Create a managed object
func (s *Service) createB(body any) *core.TryRequest {
	req := s.Service.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObjects)
	return core.NewTryRequest(s.Client, req, "")
}

// Get a managed object
func (s *Service) getB(ID string, opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}

// Update a managed object
func (s *Service) updateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}

// List of supported measurement types for a given managed object
func (s *Service) ListSupportedMeasurements(ctx context.Context, ID string) op.Result[jsonmodels.SupportedMeasurements] {
	return core.Execute(ctx, s.listSupportedMeasurementsB(ID), jsonmodels.NewSupportedMeasurements)
}

func (s *Service) listSupportedMeasurementsB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObjectSupportedMeasurements)
	return core.NewTryRequest(s.Client, req)
}

// List of supported measurement series for a given managed object
func (s *Service) ListSupportedSeries(ctx context.Context, ID string) op.Result[jsonmodels.SupportedSeries] {
	return core.Execute(ctx, s.listSupportedSeriesB(ID), jsonmodels.NewSupportedSeries)
}
func (s *Service) listSupportedSeriesB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObjectSupportedSeries)
	return core.NewTryRequest(s.Client, req)
}

// DeleteOptions options to delete a managed object
type DeleteOptions struct {
	// When set to true and the managed object is a device or group, all the hierarchy will be deleted
	Cascade bool `url:"cascade,omitempty"`

	// When set to true all the hierarchy will be deleted without checking the type of managed object. It takes precedence over the parameter cascade
	ForceCascade bool `url:"forceCascade,omitempty"`

	// When set to true and the managed object is a device, it deletes the associated device user (credentials)
	WithDeviceUser bool `url:"withDeviceUser,omitempty"`
}

// Delete a managed object
func (s *Service) deleteB(ID string, opt DeleteOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}

// ResolveID resolves an ID string that may contain a resolver scheme.
// If meta is not nil, it will be populated with metadata about the resolution (e.g., name, type, etc.).
// Examples:
//   - "12345" -> "12345" (plain ID, meta: {"source": "direct-id", ...})
//   - "name:device01" -> "<id>" (meta: {"name": "device01", "resolver": "name:device01", ...})
//   - "ext:c8y_Serial:ABC123" -> "<id>" (meta: {"externalType": "c8y_Serial", "externalID": "ABC123", ...})
//   - "query:name eq 'device01'" -> "<id>" (meta: {"query": "...", ...})
func (s *Service) ResolveID(ctx context.Context, id string, meta map[string]any) (string, error) {
	resolver, err := s.parseResolver(id)
	if err != nil {
		return "", err
	}
	result, err := resolver.ResolveID(ctx)
	if err != nil {
		return "", err
	}
	if meta != nil {
		for k, v := range result.Meta {
			meta[k] = v
		}
	}
	return result.ID, nil
}

// RegisterResolver allows registering custom ID resolvers for use with ResolveID
// Example: RegisterResolver("custom", myResolver)
// Then use: ResolveID(ctx, "custom:value")
func (s *Service) RegisterResolver(scheme string, resolver source.Resolver) {
	s.customResolvers[scheme] = resolver
}

// ByID creates a direct ID reference string (no lookup needed).
// Returns: "12345" or "id:12345"
// Example: service.Get(ctx, service.ByID("12345"), opts)
func (s *Service) ByID(id string) string {
	return id
}

// ByExternalID creates an external ID reference string.
// Returns: "ext:type:externalID"
// Example: service.Get(ctx, service.ByExternalID("c8y_Serial", "ABC123"), opts)
func (s *Service) ByExternalID(typ, externalID string) string {
	return externalIDResolver{
		Type:       typ,
		ExternalID: externalID,
	}.String()
}

// ByName creates a name-based lookup reference string.
// Supports wildcard patterns using "*".
// Returns: "name:deviceName"
// Example: service.Get(ctx, service.ByName("MyDevice"), opts)
func (s *Service) ByName(name string) string {
	return nameResolver{
		Name: name,
	}.String()
}

// ByQuery creates a query-based lookup reference string.
// The query should return exactly one result.
// Returns: "query:..."
// Example: service.Get(ctx, service.ByQuery("type eq 'c8y_Device'"), opts)
func (s *Service) ByQuery(query string) string {
	return queryResolver{
		Query: query,
	}.String()
}

// ManagedObjectStream provides an iterator for realtime managed object subscriptions
type ManagedObjectStream = realtime.Stream[realtime.StreamData[jsonmodels.ManagedObject]]

// SubscribeStream subscribes to realtime managed objects and returns a typed stream iterator.
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
//	streamResult := client.Inventory.ManagedObjects.SubscribeStream(ctx, deviceID)
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
//	    log.Printf("ManagedObject %s: %s (%s)", item.Action, item.Data.Name(), item.Data.ID())
//	    if item.Data.Type() == "c8y_Device" {
//	        break
//	    }
//	}
//
//	// Or using range with Seq() (simpler, errors stop iteration)
//	for item := range stream.Seq() {
//	    log.Printf("ManagedObject %s: %s (%s)", item.Action, item.Data.Name(), item.Data.ID())
//	}
//	if err := stream.Err(); err != nil {
//	    return err
//	}
func (s *Service) SubscribeStream(ctx context.Context, ID string) op.Result[*ManagedObjectStream] {
	err := s.RealtimeClient.Connect()
	if err != nil {
		return op.Failed[*ManagedObjectStream](err, false)
	}

	messages := make(chan *realtime.Message, 10)
	pattern := realtime.ManagedObjects(ID)
	errorChan := s.RealtimeClient.Subscribe(ctx, pattern, messages)
	stream := realtime.NewStream(ctx, messages, errorChan, func(msg *realtime.Message) realtime.StreamData[jsonmodels.ManagedObject] {
		return realtime.StreamData[jsonmodels.ManagedObject]{
			Action:  msg.Payload.RealtimeAction,
			Channel: msg.Channel,
			Data:    jsonmodels.NewManagedObject(msg.Payload.Data.Bytes()),
		}
	}, func() {
		// Cleanup: unsubscribe from the realtime channel
		s.RealtimeClient.Unsubscribe(pattern)
	})

	return op.OK(stream)
}
