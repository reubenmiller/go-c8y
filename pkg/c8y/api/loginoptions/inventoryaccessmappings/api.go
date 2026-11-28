// Package inventoryaccessmappings provides access to a login option's inventory access
// mappings (/tenant/loginOptions/{typeOrId}/inventoryAccessMappings), which map a
// condition to the inventory-role assignments granted to a user on SSO login.
package inventoryaccessmappings

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var apiInventoryAccessMappings = "/tenant/loginOptions/{typeOrId}/inventoryAccessMappings"
var apiInventoryAccessMapping = "/tenant/loginOptions/{typeOrId}/inventoryAccessMappings/{id}"

const (
	paramTypeOrID = "typeOrId"
	paramID       = "id"
)

const resultProperty = "inventoryAccessMappings"

// Service provides access to a login option's inventory access mappings.
type Service struct{ svc httpcore.Service }

func NewService(s *httpcore.Service) *Service {
	return &Service{svc: *s}
}

// List returns all inventory access mappings for a login option (identified by type or id).
func (s *Service) List(ctx context.Context, typeOrID string) op.Result[jsonmodels.InventoryAccessMapping] {
	return httpcore.ExecuteCollection(ctx, s.listB(typeOrID), resultProperty, "", jsonmodels.NewInventoryAccessMapping)
}

func (s *Service) listB(typeOrID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramTypeOrID, typeOrID).
		SetURL(apiInventoryAccessMappings)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

// Get retrieves a single inventory access mapping by id.
func (s *Service) Get(ctx context.Context, typeOrID, id string) op.Result[jsonmodels.InventoryAccessMapping] {
	return httpcore.Execute(ctx, s.getB(typeOrID, id), jsonmodels.NewInventoryAccessMapping)
}

func (s *Service) getB(typeOrID, id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramTypeOrID, typeOrID).
		SetPathParam(paramID, id).
		SetURL(apiInventoryAccessMapping)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Create adds a new inventory access mapping to a login option.
func (s *Service) Create(ctx context.Context, typeOrID string, body any) op.Result[jsonmodels.InventoryAccessMapping] {
	return httpcore.Execute(ctx, s.createB(typeOrID, body), jsonmodels.NewInventoryAccessMapping)
}

func (s *Service) createB(typeOrID string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetPathParam(paramTypeOrID, typeOrID).
		SetBody(body).
		SetURL(apiInventoryAccessMappings)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Update modifies an existing inventory access mapping.
func (s *Service) Update(ctx context.Context, typeOrID, id string, body any) op.Result[jsonmodels.InventoryAccessMapping] {
	return httpcore.Execute(ctx, s.updateB(typeOrID, id, body), jsonmodels.NewInventoryAccessMapping)
}

func (s *Service) updateB(typeOrID, id string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetPathParam(paramTypeOrID, typeOrID).
		SetPathParam(paramID, id).
		SetBody(body).
		SetURL(apiInventoryAccessMapping)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Delete removes an inventory access mapping.
func (s *Service) Delete(ctx context.Context, typeOrID, id string) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.deleteB(typeOrID, id))
}

func (s *Service) deleteB(typeOrID, id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(paramTypeOrID, typeOrID).
		SetPathParam(paramID, id).
		SetURL(apiInventoryAccessMapping)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
