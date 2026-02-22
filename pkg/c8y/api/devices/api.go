package devices

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/devices/enrollment"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/devices/registration"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects/childadditions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects/childassets"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects/childdevices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiDeviceControlAccessToken = "/devicecontrol/deviceAccessToken"

const ParamId = "id"

const ResultProperty = managedobjects.ResultProperty

func NewService(s *core.Service) *Service {
	mos := managedobjects.NewService(s)
	return &Service{
		Service:        *s,
		Enrollment:     enrollment.NewService(s),
		Registration:   registration.NewService(s),
		ChildAdditions: mos.ChildAdditions,
		ChildAssets:    mos.ChildAssets,
		ChildDevices:   mos.ChildDevices,
		managedObjects: *mos,
	}
}

// Service inventory api to interact with managed objects
// type Service core.Service
type Service struct {
	core.Service

	Enrollment     *enrollment.Service
	Registration   *registration.Service
	ChildAdditions *childadditions.Service
	ChildAssets    *childassets.Service
	ChildDevices   *childdevices.Service

	managedObjects managedobjects.Service
}

// ListOptions filter managed object
type ListOptions struct {
	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	Text string `url:"text,omitempty"`

	// Read-only collection of managed objects fetched for a given list of ids (placeholder {ids}),for example "?ids=41,43,68".
	Ids []string `url:"ids,omitempty"`

	Query string `url:"q,omitempty"`

	managedobjects.GetOptions

	// Pagination options
	pagination.PaginationOptions
}

// ManagedObjectIterator provides iteration over managed objects
type ManagedObjectIterator = pagination.Iterator[jsonmodels.ManagedObject]

// List managed objects
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteCollection(ctx, s.listB(opt), managedobjects.ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

// ListAll returns an iterator for all devices
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *ManagedObjectIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.ManagedObject] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewManagedObject,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(managedobjects.ApiManagedObjects)
	return core.NewTryRequest(s.Client, req, managedobjects.ResultProperty)
}

// FindOptions filter devices
type FindOptions struct {
	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	// Read-only collection of managed objects fetched for a given list of ids (placeholder {ids}),for example "?ids=41,43,68".
	Ids []string `url:"ids,omitempty"`

	Query string `url:"q,omitempty"`

	managedobjects.GetOptions

	// Pagination options
	pagination.PaginationOptions
}

// Find managed objects
func (s *Service) Find(ctx context.Context, opt FindOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteCollection(ctx, s.findB(opt), managedobjects.ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

func (s *Service) findB(opt FindOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(managedobjects.ApiManagedObjects)
	return core.NewTryRequest(s.Client, req, managedobjects.ResultProperty)
}

// CreateDevice creates a new device with the given name.
// It automatically includes the c8y_IsDevice fragment.
// For more control, use jsonmodels.NewDevice() or jsonmodels.NewDeviceWithType() and pass to Create().
func (s *Service) CreateDevice(ctx context.Context, name string) op.Result[jsonmodels.ManagedObject] {
	return s.managedObjects.Create(ctx, jsonmodels.NewDevice(name))
}

// Create creates a new managed object
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.ManagedObject] {
	return s.managedObjects.Create(ctx, body)
}

// Get retrieves a device by ID
func (s *Service) Get(ctx context.Context, ID string, opt managedobjects.GetOptions) op.Result[jsonmodels.ManagedObject] {
	return s.managedObjects.Get(ctx, ID, opt)
}

// Update updates a device
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.ManagedObject] {
	return s.managedObjects.Update(ctx, ID, body)
}

// DeleteOptions options for deleting a managed object
type DeleteOptions = managedobjects.DeleteOptions

// Delete deletes a device by ID
func (s *Service) Delete(ctx context.Context, ID string, opt DeleteOptions) op.Result[core.NoContent] {
	return s.managedObjects.Delete(ctx, ID, opt)
}

// GetOrCreateByName searches by name and optionally type, creating if not found
func (s *Service) GetOrCreateByName(ctx context.Context, name, objType string, body map[string]any) op.Result[jsonmodels.ManagedObject] {
	return s.managedObjects.GetOrCreateByName(ctx, name, objType, body)
}

// GetOrCreateByFragment searches for objects with a specific fragment property
func (s *Service) GetOrCreateByFragment(ctx context.Context, fragment string, body map[string]any) op.Result[jsonmodels.ManagedObject] {
	return s.managedObjects.GetOrCreateByFragment(ctx, fragment, body)
}

// GetOrCreateWith provides a generic query-based lookup
// Example queries:
//   - "name eq 'device01' and type eq 'c8y_Device'"
//   - "has(c8y_IsDevice) and c8y_Serial eq '12345'"
//   - "fragmentType eq 'c8y_CustomFragment'"
func (s *Service) GetOrCreateWith(ctx context.Context, body map[string]any, query string) op.Result[jsonmodels.ManagedObject] {
	return s.managedObjects.GetOrCreateWith(ctx, body, query)
}

// GetOrCreateByExternalIDOptions options for GetOrCreateByExternalID
type GetOrCreateByExternalIDOptions = managedobjects.GetOrCreateByExternalIDOptions

// GetOrCreateByExternalID looks up a managed object by external identity,
// creating both the managed object and identity if not found.
func (s *Service) GetOrCreateByExternalID(ctx context.Context, opts GetOrCreateByExternalIDOptions) op.Result[jsonmodels.ManagedObject] {
	return s.managedObjects.GetOrCreateByExternalID(ctx, opts)
}

// ListSupportedMeasurements gets supported measurement types for a device
func (s *Service) ListSupportedMeasurements(ctx context.Context, ID string) op.Result[jsonmodels.SupportedMeasurements] {
	return s.managedObjects.ListSupportedMeasurements(ctx, ID)
}

// ListSupportedSeries gets supported series for a device
func (s *Service) ListSupportedSeries(ctx context.Context, ID string) op.Result[jsonmodels.SupportedSeries] {
	return s.managedObjects.ListSupportedSeries(ctx, ID)
}
