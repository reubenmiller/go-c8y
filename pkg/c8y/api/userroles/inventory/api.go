// Package inventoryroles provides a client service for the Cumulocity
// Inventory Roles API (/user/inventoryroles).
//
// An inventory role is a scoped permission set that limits a user's or
// group's access to specific managed-object types, operations, or tenants.
// Inventory roles can be assigned to users directly via the Device Permissions
// API or to user groups.
//
// Required roles:
//   - ROLE_USER_MANAGEMENT_READ  - for read access (List, Get)
//   - ROLE_USER_MANAGEMENT_ADMIN - for write access (Create, Update, Delete)
package inventoryroles

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// apiInventoryRoles is the collection endpoint for inventory roles.
var apiInventoryRoles = "/user/inventoryroles"

// apiInventoryRole is the single-item endpoint for inventory roles.
var apiInventoryRole = "/user/inventoryroles/{id}"

// paramID is the path-parameter name used in apiInventoryRole.
var paramID = "id"

// resultProperty is the JSON key that wraps the array of inventory roles
// in a collection response.
const resultProperty = "roles"

// NewService creates a new InventoryRoles service backed by the provided
// httpcore.Service (HTTP client + tenant configuration).
func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc: *s,
	}
}

// Service provides access to the Cumulocity Inventory Roles API.
type Service struct {
	svc httpcore.Service
}

// ListOptions controls pagination of an inventory-role list request.
type ListOptions struct {
	pagination.PaginationOptions
}

// InventoryRoleIterator is a lazy iterator over a (potentially multi-page)
// collection of inventory roles.
type InventoryRoleIterator = pagination.Iterator[jsonmodels.InventoryRole]

// List returns a single page of inventory roles.
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.InventoryRole] {
	return httpcore.ExecuteCollection(ctx, s.listB(opt), resultProperty, types.ResponseFieldStatistics, jsonmodels.NewInventoryRole)
}

// ListAll returns a lazy iterator that transparently pages through all
// inventory roles matching the given options.
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *InventoryRoleIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.InventoryRole] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewInventoryRole,
	)
}

func (s *Service) listB(opt any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiInventoryRoles)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

// Get retrieves a single inventory role by its ID.
func (s *Service) Get(ctx context.Context, id string) op.Result[jsonmodels.InventoryRole] {
	return httpcore.Execute(ctx, s.getB(id), jsonmodels.NewInventoryRole)
}

func (s *Service) getB(id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramID, id).
		SetURL(apiInventoryRole)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Create submits a new inventory role. The body must contain at minimum a
// name. An optional description and permissions array may also be supplied.
//
// Any serialisable value (struct, map[string]any, etc.) is accepted.
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.InventoryRole] {
	return httpcore.Execute(ctx, s.createB(body), jsonmodels.NewInventoryRole)
}

func (s *Service) createB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(apiInventoryRoles)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Update modifies an existing inventory role. Updatable fields include
// name, description and permissions.
func (s *Service) Update(ctx context.Context, id string, body any) op.Result[jsonmodels.InventoryRole] {
	return httpcore.Execute(ctx, s.updateB(id, body), jsonmodels.NewInventoryRole)
}

func (s *Service) updateB(id string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(paramID, id).
		SetBody(body).
		SetURL(apiInventoryRole)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Delete removes an inventory role by its ID. A 204 No Content
// response indicates success.
func (s *Service) Delete(ctx context.Context, id string) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.deleteB(id)).IgnoreNotFound()
}

func (s *Service) deleteB(id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(paramID, id).
		SetURL(apiInventoryRole)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
