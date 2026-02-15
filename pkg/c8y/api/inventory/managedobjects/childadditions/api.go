package childadditions

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects/child"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"resty.dev/v3"
)

var ApiManagedObjectChildAdditions = "/inventory/managedObjects/{id}/childAdditions"
var ApiManagedObjectChildAddition = "/inventory/managedObjects/{id}/childAdditions/{child}"

const ParamId = "id"
const ParamChild = "child"

const ResultProperty = "references.#.managedObject"

// Service
type Service core.Service

func NewService(common *core.Service) *Service {
	return (*Service)(common)
}

type ListOptions child.ListOptions

// ManagedObjectIterator provides iteration over managed objects
type ManagedObjectIterator = pagination.Iterator[jsonmodels.ManagedObject]

// List child additions of a parent
func (s *Service) List(ctx context.Context, parentID string, opt ListOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteCollection(ctx, s.listB(parentID, opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

// ListAll returns an iterator for all child additions
func (s *Service) ListAll(ctx context.Context, parentID string, opts ListOptions) *ManagedObjectIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.ManagedObject] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, parentID, o)
		},
		jsonmodels.NewManagedObject,
	)
}

func (s *Service) listB(parentID string, opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, parentID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiManagedObjectChildAdditions)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get existing child addition from a parent
func (s *Service) Get(ctx context.Context, parentID string, childID string) op.Result[jsonmodels.ManagedObject] {
	return core.Execute(ctx, s.getB(parentID, childID), jsonmodels.NewManagedObject)
}

func (s *Service) getB(parentID string, childID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, parentID).
		SetPathParam(ParamChild, childID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObjectChildAddition)
	return core.NewTryRequest(s.Client, req)
}

// Create a new managed object and assign it as a child addition to an existing managed object
func (s *Service) Create(ctx context.Context, parentID string, body any) op.Result[jsonmodels.ManagedObject] {
	return core.Execute(ctx, s.createB(parentID, body), jsonmodels.NewManagedObject)
}

func (s *Service) createB(parentID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamId, parentID).
		SetContentType(types.MimeTypeManagedObject).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiManagedObjectChildAdditions)
	return core.NewTryRequest(s.Client, req)
}

// Assign an existing child addition to a managed object
func (s *Service) Assign(ctx context.Context, parentID string, child any) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.assignB(parentID, child))
}

func (s *Service) assignB(parentID string, child any) *core.TryRequest {
	contentType, body := model.FromManagedObjectChildReferences(child)
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetContentType(contentType).
		SetBody(body).
		SetPathParam(ParamId, parentID).
		SetURL(ApiManagedObjectChildAdditions)
	return core.NewTryRequest(s.Client, req)
}

// Unassign a child addition from a managed object
func (s *Service) Unassign(ctx context.Context, parentID string, child any) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.unassignB(parentID, child))
}

func (s *Service) unassignB(parentID string, child any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetContentType(types.MimeTypeManagedObjectCollection).
		SetBody(model.ToManagedObjectChildReferences(child)).
		SetPathParam(ParamId, parentID).
		SetURL(ApiManagedObjectChildAdditions)
	return core.NewTryRequest(s.Client, req)
}
