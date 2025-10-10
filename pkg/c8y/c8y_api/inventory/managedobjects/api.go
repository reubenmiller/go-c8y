package managedobjects

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects/childadditions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects/childassets"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects/childdevices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"resty.dev/v3"
)

var ApiManagedObjects = "/inventory/managedObjects"
var ApiManagedObject = "/inventory/managedObjects/{id}"
var ApiManagedObjectSupportedMeasurements = "/inventory/managedObjects/{id}/supportedMeasurements"
var ApiManagedObjectSupportedSeries = "/inventory/managedObjects/{id}/supportedSeries"

const ParamId = "id"

func NewService(s *core.Service) *Service {
	return &Service{
		Service:        *s,
		ChildAdditions: childadditions.NewService(s),
	}
}

// Service inventory api to interact with managed objects
// type Service core.Service
type Service struct {
	core.Service
	ChildAdditions *childadditions.Service
	ChildAssets    *childassets.Service
	ChildDevices   *childdevices.Service
}

// ListOptions filter managed object
type ListOptions struct {
	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	Text string `url:"text,omitempty"`

	// Read-only collection of managed objects fetched for a given list of ids (placeholder {ids}),for example "?ids=41,43,68".
	Ids []string `url:"ids,omitempty"`

	Query string `url:"query,omitempty"`

	*GetOptions

	// Pagination options
	pagination.PaginationOptions
}

// List managed objects
func (s *Service) List(ctx context.Context, opt ListOptions) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiManagedObjects)
}

type GetOptions struct {
	WithParents       bool `url:"withParents,omitempty"`
	WithChildren      bool `url:"withChildren,omitempty"`
	withChildrenCount bool `url:"withChildrenCount,omitempty"`
	SkipChildrenNames bool `url:"skipChildrenNames,omitempty"`
	WithLatestValues  bool `url:"withLatestValues,omitempty"`
}

// Create a managed object
func (s *Service) Create(ctx context.Context, body any) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetURL(ApiManagedObject)
}

// Get a managed object
func (s *Service) Get(ctx context.Context, ID string, opt GetOptions) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiManagedObject)
}

// Update a managed object
func (s *Service) Update(ctx context.Context, ID string, body any) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetBody(body).
		SetURL(ApiManagedObject)
}

// List of supported measurement types for a given managed object
func (s *Service) ListSupportedMeasurements(ctx context.Context, ID string) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiManagedObjectSupportedMeasurements)
}

// List of supported measurement series for a given managed object
func (s *Service) ListSupportedSeries(ctx context.Context, ID string) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiManagedObjectSupportedSeries)
}
