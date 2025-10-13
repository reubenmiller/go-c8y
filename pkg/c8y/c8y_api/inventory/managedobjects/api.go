package managedobjects

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects/childadditions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects/childassets"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects/childdevices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
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
	return &Service{
		Service:        *s,
		ChildAdditions: childadditions.NewService(s),
		ChildAssets:    childassets.NewService(s),
		ChildDevices:   childdevices.NewService(s),
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

	GetOptions

	// Pagination options
	pagination.PaginationOptions
}

// List managed objects
func (s *Service) List(ctx context.Context, opt ListOptions) (*model.ManagedObjectCollection, error) {
	return core.ExecuteResultOnly[model.ManagedObjectCollection](ctx, s.ListB(opt))
}

func (s *Service) ListB(opt any) *core.TryRequest {
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
func (s *Service) Create(ctx context.Context, body any) (*model.ManagedObject, error) {
	return core.ExecuteResultOnly[model.ManagedObject](ctx, s.CreateB(body))
}

func (s *Service) CreateB(body any) *core.TryRequest {
	req := s.Service.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObjects)
	return core.NewTryRequest(s.Client, req, "")
}

// Get a managed object
func (s *Service) Get(ctx context.Context, ID string, opt GetOptions) (*model.ManagedObject, error) {
	return core.ExecuteResultOnly[model.ManagedObject](ctx, s.GetB(ID, opt))
}

func (s *Service) GetB(ID string, opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}

// Update a managed object
func (s *Service) Update(ctx context.Context, ID string, body any) (*model.ManagedObject, error) {
	return core.ExecuteResultOnly[model.ManagedObject](ctx, s.UpdateB(ID, body))
}

func (s *Service) UpdateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}

// List of supported measurement types for a given managed object
func (s *Service) ListSupportedMeasurements(ctx context.Context, ID string) (*model.SupportedMeasurements, error) {
	return core.ExecuteResultOnly[model.SupportedMeasurements](ctx, s.ListSupportedMeasurementsB(ID))
}

func (s *Service) ListSupportedMeasurementsB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObjectSupportedMeasurements)
	return core.NewTryRequest(s.Client, req)
}

// List of supported measurement series for a given managed object
func (s *Service) ListSupportedSeries(ctx context.Context, ID string) (*model.SupportedSeries, error) {
	return core.ExecuteResultOnly[model.SupportedSeries](ctx, s.ListSupportedSeriesB(ID))
}
func (s *Service) ListSupportedSeriesB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObjectSupportedSeries)
	return core.NewTryRequest(s.Client, req)
}
