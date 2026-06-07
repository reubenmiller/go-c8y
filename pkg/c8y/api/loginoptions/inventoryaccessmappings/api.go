// Package inventoryaccessmappings provides access to a login option's inventory access
// mappings (/tenant/loginOptions/{typeOrId}/inventoryAccessMappings), which map a
// condition to the inventory-role assignments granted to a user on SSO login.
package inventoryaccessmappings

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiInventoryAccessMappings = "/tenant/loginOptions/{typeOrId}/inventoryAccessMappings"
var ApiInventoryAccessMapping = "/tenant/loginOptions/{typeOrId}/inventoryAccessMappings/{id}"

const (
	ParamTypeOrID = "typeOrId"
	ParamID       = "id"
)

const ResultProperty = "inventoryAccessMappings"

// Service provides access to a login option's inventory access mappings.
type Service struct{ core.Service }

func NewService(s *core.Service) *Service {
	return &Service{Service: *s}
}

// List returns all inventory access mappings for a login option (identified by type or id).
func (s *Service) List(ctx context.Context, typeOrID string) op.Result[jsonmodels.InventoryAccessMapping] {
	return core.ExecuteCollection(ctx, s.listB(typeOrID), ResultProperty, "", jsonmodels.NewInventoryAccessMapping)
}

func (s *Service) listB(typeOrID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTypeOrID, typeOrID).
		SetURL(ApiInventoryAccessMappings)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get retrieves a single inventory access mapping by id.
func (s *Service) Get(ctx context.Context, typeOrID, id string) op.Result[jsonmodels.InventoryAccessMapping] {
	return core.Execute(ctx, s.getB(typeOrID, id), jsonmodels.NewInventoryAccessMapping)
}

func (s *Service) getB(typeOrID, id string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTypeOrID, typeOrID).
		SetPathParam(ParamID, id).
		SetURL(ApiInventoryAccessMapping)
	return core.NewTryRequest(s.Client, req)
}

// Create adds a new inventory access mapping to a login option.
func (s *Service) Create(ctx context.Context, typeOrID string, body any) op.Result[jsonmodels.InventoryAccessMapping] {
	return core.Execute(ctx, s.createB(typeOrID, body), jsonmodels.NewInventoryAccessMapping)
}

func (s *Service) createB(typeOrID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetPathParam(ParamTypeOrID, typeOrID).
		SetBody(body).
		SetURL(ApiInventoryAccessMappings)
	return core.NewTryRequest(s.Client, req)
}

// Update modifies an existing inventory access mapping.
func (s *Service) Update(ctx context.Context, typeOrID, id string, body any) op.Result[jsonmodels.InventoryAccessMapping] {
	return core.Execute(ctx, s.updateB(typeOrID, id, body), jsonmodels.NewInventoryAccessMapping)
}

func (s *Service) updateB(typeOrID, id string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetPathParam(ParamTypeOrID, typeOrID).
		SetPathParam(ParamID, id).
		SetBody(body).
		SetURL(ApiInventoryAccessMapping)
	return core.NewTryRequest(s.Client, req)
}

// Delete removes an inventory access mapping.
func (s *Service) Delete(ctx context.Context, typeOrID, id string) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(typeOrID, id))
}

func (s *Service) deleteB(typeOrID, id string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamTypeOrID, typeOrID).
		SetPathParam(ParamID, id).
		SetURL(ApiInventoryAccessMapping)
	return core.NewTryRequest(s.Client, req)
}
