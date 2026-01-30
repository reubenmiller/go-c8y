package childadditions

import (
	"context"

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

// List child additions of a parent
func (s *Service) List(ctx context.Context, parentID string, opt ListOptions) (*model.ManagedObjectCollection, error) {
	return core.ExecuteResultOnly[model.ManagedObjectCollection](ctx, s.ListB(parentID, opt))
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
func (s *Service) Get(ctx context.Context, parentID string, childID string) (*model.ManagedObject, error) {
	return core.ExecuteResultOnly[model.ManagedObject](ctx, s.GetB(parentID, childID))
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
func (s *Service) Create(ctx context.Context, parentID string, body any) (*model.ManagedObject, error) {
	return core.ExecuteResultOnly[model.ManagedObject](ctx, s.CreateB(parentID, body))
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
