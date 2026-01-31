package childadditions

import (
	"context"
	"iter"
	"log/slog"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects/child"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiManagedObjectChildAdditions = "/inventory/managedObjects/{id}/childAdditions"
var ApiManagedObjectChildAddition = "/inventory/managedObjects/{id}/childAdditions/{child}"

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
type ManagedObjectIterator struct {
	items iter.Seq[jsonmodels.ManagedObject]
	err   error
}

func (it *ManagedObjectIterator) Items() iter.Seq[jsonmodels.ManagedObject] {
	return it.items
}

func (it *ManagedObjectIterator) Err() error {
	return it.err
}

func paginateManagedObjects(ctx context.Context, fetch func(page int) op.Result[jsonmodels.ManagedObject], maxItems int64) *ManagedObjectIterator {
	iterator := &ManagedObjectIterator{}

	iterator.items = func(yield func(jsonmodels.ManagedObject) bool) {
		page := 1
		count := int64(0)
		for {
			result := fetch(page)
			if result.Err != nil {
				iterator.err = result.Err
				return
			}
			countBeforeResults := count
			for doc := range result.Data.Iter() {
				if maxItems > 0 && count >= maxItems {
					return
				}
				item := jsonmodels.NewManagedObject(doc.Bytes())
				if !yield(item) {
					return
				}
				count++
			}
			if countBeforeResults == count {
				slog.Info("Stopping pagination as results array is empty")
				return
			}

			totalPages, ok := result.Meta["totalPages"].(int64)
			if ok && page >= int(totalPages) {
				return
			}
			page++
		}
	}

	return iterator
}

// List child additions of a parent
func (s *Service) List(ctx context.Context, parentID string, opt ListOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnCollection(ctx, s.ListB(parentID, opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

// ListAll returns an iterator for all child additions
func (s *Service) ListAll(ctx context.Context, parentID string, opts ListOptions) *ManagedObjectIterator {
	if opts.PageSize == 0 {
		opts.PageSize = 2000
	}
	return paginateManagedObjects(ctx, func(page int) op.Result[jsonmodels.ManagedObject] {
		opts.CurrentPage = page
		return s.List(ctx, parentID, opts)
	}, opts.GetMaxItems())
}

func (s *Service) ListB(parentID string, opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, parentID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiManagedObjectChildAdditions)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get existing child addition from a parent
func (s *Service) Get(ctx context.Context, parentID string, childID string) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnResult(ctx, s.GetB(parentID, childID), jsonmodels.NewManagedObject)
}

func (s *Service) GetB(parentID string, childID string) *core.TryRequest {
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
	return core.ExecuteReturnResult(ctx, s.CreateB(parentID, body), jsonmodels.NewManagedObject)
}

func (s *Service) CreateB(parentID string, body any) *core.TryRequest {
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
		SetURL(ApiManagedObjectChildAdditions)
	return core.NewTryRequest(s.Client, req)
}

// Unassign a child addition from a managed object
func (s *Service) Unassign(ctx context.Context, parentID string, child any) error {
	return core.ExecuteNoResult(ctx, s.UnassignB(parentID, child))
}

func (s *Service) UnassignB(parentID string, child any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetContentType(types.MimeTypeManagedObjectCollection).
		SetBody(model.ToManagedObjectChildReferences(child)).
		SetPathParam(ParamId, parentID).
		SetURL(ApiManagedObjectChildAdditions)
	return core.NewTryRequest(s.Client, req)
}
