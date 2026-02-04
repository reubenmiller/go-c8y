package managedobjects

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/identity"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects/childadditions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects/childassets"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects/childdevices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/source"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
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
		ChildAdditions:  childadditions.NewService(s),
		ChildAssets:     childassets.NewService(s),
		ChildDevices:    childdevices.NewService(s),
	}

	// Create the source builder with lookups
	service.sourceBuilder = source.NewBuilder(
		// External ID lookup
		func(ctx context.Context, typ, extID string) (string, map[string]any, error) {
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
		},
		// Name lookup
		func(ctx context.Context, name string) (string, map[string]any, error) {
			opts := ListOptions{}
			opts.PaginationOptions.PageSize = 1
			opts.Query = model.NewInventoryQuery().
				AddFilterEqStr("name", name).
				// AddFilterPart(additionalQueries...).
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
		},
		// Query lookup
		func(ctx context.Context, query string) (string, map[string]any, error) {
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
		},
	)

	return service
}

// Service inventory api to interact with managed objects
// type Service core.Service
type Service struct {
	core.Service
	identityService *identity.Service
	ChildAdditions  *childadditions.Service
	ChildAssets     *childassets.Service
	ChildDevices    *childdevices.Service
	sourceBuilder   *source.Builder
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

// Source Resolution Convenience Methods
// These methods provide a more discoverable way to create source resolvers
// for managed objects, while still returning the generic source.Resolver interface.

// ByID creates a resolver for a managed object by its direct ID.
// Returns a source.Resolver that can be used with any API that accepts source resolution.
func (s *Service) ByID(id string) source.Resolver {
	return source.ID(id)
}

// ByExternalID creates a resolver that looks up a managed object by its external ID.
// The lookup will be performed when ResolveID() is called on the returned resolver.
// Returns a source.Resolver that can be used with any API that accepts source resolution.
func (s *Service) ByExternalID(typ, externalID string) source.Resolver {
	return source.ExternalID{
		Type:       typ,
		ExternalID: externalID,
		Lookup: func(ctx context.Context, t, extID string) (string, map[string]any, error) {
			result := s.identityService.Get(ctx, identity.IdentityOptions{
				Type:       t,
				ExternalID: extID,
			})
			if result.Err != nil {
				return "", nil, result.Err
			}
			// Return metadata about the resolved object
			meta := map[string]any{
				"externalType": t,
				"externalID":   extID,
			}
			return result.Data.ManagedObjectID(), meta, nil
		},
	}
}

// ByName creates a resolver that looks up a managed object by its name.
func (s *Service) ByName(name string, additionalQueries ...string) source.Resolver {
	return source.Name{
		Name: name,
		Lookup: func(ctx context.Context, n string) (string, map[string]any, error) {
			result := s.List(context.Background(), ListOptions{
				DeviceQuery: model.NewInventoryQuery().
					AddFilterEqStr("name", n).
					AddFilterPart(additionalQueries...).
					AddOrderBy("name").
					AddOrderBy("creationTime").
					Build(),
				PaginationOptions: pagination.PaginationOptions{
					PageSize: 1,
				},
			})
			if result.Err != nil {
				return "", nil, result.Err
			}

			// if result.Data.Length() == 0 {
			// 	return "", nil, fmt.Errorf("no device found with name: %s", n)
			// }

			for item := range op.Iter(result) {
				meta := map[string]any{
					"name":  item.Name(),
					"owner": item.Owner(),
				}
				return item.ID(), meta, nil
			}

			return "", nil, fmt.Errorf("no device found with name: %s", n)
		},
	}
}

// Custom creates a resolver with custom resolution logic.
// This allows you to define your own logic for resolving a managed object ID.
// Returns a source.Resolver that can be used with any API that accepts source resolution.
func (s *Service) Custom(description string, resolve func(context.Context) (string, map[string]any, error)) source.Resolver {
	return source.Custom{
		Description: description,
		Resolve:     resolve,
	}
}

// ResolveID resolves an ID string that may contain a resolver scheme.
// If meta is not nil, it will be populated with metadata about the resolution (e.g., name, type, etc.).
// Examples:
//   - "12345" -> "12345" (plain ID, meta: {"source": "direct-id", ...})
//   - "name:device01" -> "<id>" (meta: {"name": "device01", "resolver": "name:device01", ...})
//   - "ext:c8y_Serial:ABC123" -> "<id>" (meta: {"externalType": "c8y_Serial", "externalID": "ABC123", ...})
//   - "query:name eq 'device01'" -> "<id>" (meta: {"query": "...", ...})
func (s *Service) ResolveID(ctx context.Context, id string, meta map[string]any) (string, error) {
	resolver, err := s.sourceBuilder.Parse(id)
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
	s.sourceBuilder.RegisterResolver(scheme, resolver)
}
