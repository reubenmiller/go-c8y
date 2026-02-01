package childassets

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects/child"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiManagedObjectChildAssets = "/inventory/managedObjects/{id}/childAssets"
var ApiManagedObjectChildAsset = "/inventory/managedObjects/{id}/childAssets/{child}"

const ParamId = "id"
const ParamChild = "child"

const ResultProperty = "managedObjects"

// Service
type Service core.Service

func NewService(common *core.Service) *Service {
	return (*Service)(common)
}

type ListOptions child.ListOptions

// ManagedObjectIterator provides iteration over managed objects
type ManagedObjectIterator = pagination.Iterator[jsonmodels.ManagedObject]

// List child assets of a parent
func (s *Service) List(ctx context.Context, parentID string, opt ListOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnCollection(ctx, s.ListB(parentID, opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

// ListAll returns an iterator for all child assets
func (s *Service) ListAll(ctx context.Context, parentID string, opts ListOptions) *ManagedObjectIterator {
	return pagination.Paginate(ctx, opts.PaginationOptions, func() op.Result[jsonmodels.ManagedObject] {
		return s.List(ctx, parentID, opts)
	}, jsonmodels.NewManagedObject)
}

func (s *Service) ListB(parentID string, opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(parentID, parentID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiManagedObjectChildAssets)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get existing child asset from a parent
func (s *Service) Get(ctx context.Context, parentID string, childID string) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnResult(ctx, s.GetB(parentID, childID), jsonmodels.NewManagedObject)
}

func (s *Service) GetB(parentID string, childID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, parentID).
		SetPathParam(ParamChild, childID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObjectChildAsset)
	return core.NewTryRequest(s.Client, req)
}

// Create a new child asset and assign it to an existing managed object
func (s *Service) Create(ctx context.Context, parentID string, body any) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnResult(ctx, s.CreateB(parentID, body), jsonmodels.NewManagedObject)
}

func (s *Service) CreateB(parentID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamId, parentID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiManagedObjectChildAssets)
	return core.NewTryRequest(s.Client, req)
}

// Assign an existing child asset to a managed object
func (s *Service) Assign(ctx context.Context, parentID string, child any) error {
	return core.ExecuteNoResult(ctx, s.AssignB(parentID, child))
}

func (s *Service) AssignB(parentID string, child any) *core.TryRequest {
	contentType, body := model.FromManagedObjectChildReferences(child)
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetContentType(contentType).
		SetBody(body).
		SetPathParam(ParamId, parentID).
		SetURL(ApiManagedObjectChildAssets)
	return core.NewTryRequest(s.Client, req)
}

// Unassign a child asset from a managed object
func (s *Service) Unassign(ctx context.Context, parentID string, child any) error {
	return core.ExecuteNoResult(ctx, s.UnassignB(parentID, child))
}

func (s *Service) UnassignB(parentID string, child any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetContentType(types.MimeTypeManagedObjectCollection).
		SetBody(model.ToManagedObjectChildReferences(child)).
		SetPathParam(ParamId, parentID).
		SetURL(ApiManagedObjectChildAssets)
	return core.NewTryRequest(s.Client, req)
}
