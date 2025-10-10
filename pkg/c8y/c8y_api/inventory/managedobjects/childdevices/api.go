package childdevices

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects/child"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"resty.dev/v3"
)

var ApiManagedObjectChildDevices = "/inventory/managedObjects/{id}/childDevices"
var ApiManagedObjectChildDevice = "/inventory/managedObjects/{id}/childAdditions/{child}"

const ParamId = "id"
const ParamChild = "child"

// Service
type Service core.Service

func NewService(common *core.Service) *Service {
	return (*Service)(common)
}

type ListOptions child.ListOptions

// List child additions of a parent
func (s *Service) List(ctx context.Context, parentID string, opts ListOptions) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, parentID).
		SetQueryParamsFromValues(core.QueryParameters(opts)).
		SetURL(ApiManagedObjectChildDevices)
}

// Get existing child addition from a parent
func (s *Service) Get(ctx context.Context, parentID string, childID string) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, parentID).
		SetPathParam(ParamChild, childID).
		SetURL(ApiManagedObjectChildDevice)
}

// Create a new child addition and assign it to an existing managed object
func (s *Service) Create(ctx context.Context, parentID string, body any) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetPathParam(ParamId, parentID).
		SetURL(ApiManagedObjectChildDevices)
}

// Assign an existing child addition to a managed object
func (s *Service) Assign(ctx context.Context, parentID string, child any) *resty.Request {
	contentType, body := model.FromManagedObjectChildReferences(child)
	return s.Client.R().
		SetMethod(resty.MethodPost).
		SetContentType(contentType).
		SetBody(body).
		SetPathParam(ParamId, parentID).
		SetURL(ApiManagedObjectChildDevices)
}

// Unassign a child addition from a managed object
func (s *Service) Unassign(ctx context.Context, parentID string, child any) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodDelete).
		SetContentType(model.MimeTypeManagedObjectCollection).
		SetBody(model.ToManagedObjectChildReferences(child)).
		SetPathParam(ParamId, parentID).
		SetURL(ApiManagedObjectChildDevices)
}
