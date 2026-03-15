// Package devicepermissions provides a client service for the Cumulocity
// Device Permissions API.
//
// The package covers three related groups of endpoints:
//
//  1. Device permissions by managed object (OAS tag: Device Permissions):
//     GET  /user/devicePermissions/{id}  — who has permissions on managed object {id}
//     PUT  /user/devicePermissions/{id}  — update those permissions
//
//  2. Per-user inventory-role assignments (collection):
//     GET  /user/{tenantId}/users/{userId}/roles/inventory
//     POST /user/{tenantId}/users/{userId}/roles/inventory
//
//  3. Individual inventory-role assignment:
//     GET    /user/{tenantId}/users/{userId}/roles/inventory/{id}
//     PUT    /user/{tenantId}/users/{userId}/roles/inventory/{id}
//     DELETE /user/{tenantId}/users/{userId}/roles/inventory/{id}
//
// Required roles:
//
//   - ROLE_USER_MANAGEMENT_READ  - read access (Get*, List*)
//   - ROLE_USER_MANAGEMENT_ADMIN - write access (Assign, Update, Delete, set permissions)
package devicepermissions

import (
	"context"
	"strconv"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// Endpoint templates.
var (
	// ApiDevicePermissions is the managed-object-scoped device permissions endpoint.
	// The {id} path parameter is a managed-object ID.
	ApiDevicePermissions        = "/user/devicePermissions/{id}"
	ApiInventoryRoleAssignments = "/user/{tenantId}/users/{userId}/roles/inventory"
	ApiInventoryRoleAssignment  = "/user/{tenantId}/users/{userId}/roles/inventory/{id}"
)

// Path-parameter names.
var (
	ParamUserID = "userId"
	ParamID     = "id"
)

// ResultProperty is the JSON key wrapping the inventory-role assignment array
// in a collection response.
const ResultProperty = "inventoryAssignments"

// NewService creates a new Service backed by the provided core.Service.
func NewService(s *core.Service) *Service {
	return &Service{Service: *s}
}

// Service provides access to the Cumulocity Device Permissions API.
type Service struct {
	core.Service
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// InventoryRoleAssignmentIterator is a lazy iterator over a (potentially
// multi-page) collection of inventory-role assignments.
type InventoryRoleAssignmentIterator = pagination.Iterator[jsonmodels.InventoryRoleAssignment]

// GetDevicePermissionsOptions carries the path parameter for the
// managed-object-scoped device-permissions endpoints
// (GET/PUT /user/devicePermissions/{id}).
type GetDevicePermissionsOptions struct {
	// ManagedObjectID is the ID of the managed object whose permission owners
	// are retrieved or updated.
	ManagedObjectID string `url:"-"`
}

// UserScopedOptions carries the tenant and user path parameters shared by
// inventory-role assignment operations.
type UserScopedOptions struct {
	// TenantID is the owning tenant. Leave empty to use the client default.
	TenantID string `url:"-"`
	// UserID is the target user.
	UserID string `url:"-"`
}

// ListInventoryRoleAssignmentOptions carries path parameters and pagination for
// listing a user's inventory-role assignments.
type ListInventoryRoleAssignmentOptions struct {
	UserScopedOptions
	pagination.PaginationOptions
}

// GetInventoryRoleAssignmentOptions carries path parameters for a single
// inventory-role assignment.
type GetInventoryRoleAssignmentOptions struct {
	UserScopedOptions
	// ID is the numeric identifier of the assignment.
	ID int64 `url:"-"`
}

// ---------------------------------------------------------------------------
// Device Permissions
// ---------------------------------------------------------------------------

// GetDevicePermissions retrieves the users and groups that have device-level
// permissions on a specific managed object.
//
// Endpoint: GET /user/devicePermissions/{id}
// OAS response schema: DevicePermissionOwners
func (s *Service) GetDevicePermissions(ctx context.Context, opt GetDevicePermissionsOptions) op.Result[jsonmodels.DevicePermissionOwners] {
	return core.Execute(ctx, s.getDevicePermissionsB(opt), jsonmodels.NewDevicePermissionOwners)
}

func (s *Service) getDevicePermissionsB(opt GetDevicePermissionsOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, opt.ManagedObjectID).
		SetURL(ApiDevicePermissions)
	return core.NewTryRequest(s.Client, req)
}

// UpdateDevicePermissions updates the device-level permissions for users and
// groups on a specific managed object.
//
// Endpoint: PUT /user/devicePermissions/{id}
// OAS request schema: UpdatedDevicePermissions
//
// Example body:
//
//	map[string]any{
//	  "users": []map[string]any{{
//	    "userName": "jdoe",
//	    "devicePermissions": map[string]any{
//	      "12345": []string{"MANAGED_OBJECT:*:ADMIN"},
//	    },
//	  }},
//	  "groups": []map[string]any{{
//	    "id": "7",
//	    "devicePermissions": map[string]any{
//	      "12345": []string{"READ"},
//	    },
//	  }},
//	}
//
// The PUT returns HTTP 200 with no response body.
func (s *Service) UpdateDevicePermissions(ctx context.Context, opt GetDevicePermissionsOptions, body any) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.updateDevicePermissionsB(opt, body))
}

func (s *Service) updateDevicePermissionsB(opt GetDevicePermissionsOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, opt.ManagedObjectID).
		SetBody(body).
		SetURL(ApiDevicePermissions)
	return core.NewTryRequest(s.Client, req)
}

// ---------------------------------------------------------------------------
// Inventory Role Assignments (collection)
// ---------------------------------------------------------------------------

// ListInventoryRoleAssignments retrieves a single page of inventory-role
// assignments for the specified user.
func (s *Service) ListInventoryRoleAssignments(ctx context.Context, opt ListInventoryRoleAssignmentOptions) op.Result[jsonmodels.InventoryRoleAssignment] {
	return core.ExecuteCollection(ctx, s.listInventoryRoleAssignmentsB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewInventoryRoleAssignment)
}

// ListAllInventoryRoleAssignments returns a lazy iterator that transparently
// pages through all inventory-role assignments for the specified user.
func (s *Service) ListAllInventoryRoleAssignments(ctx context.Context, opts ListInventoryRoleAssignmentOptions) *InventoryRoleAssignmentIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.InventoryRoleAssignment] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.ListInventoryRoleAssignments(ctx, o)
		},
		jsonmodels.NewInventoryRoleAssignment,
	)
}

func (s *Service) listInventoryRoleAssignmentsB(opt ListInventoryRoleAssignmentOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(ParamUserID, opt.UserID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiInventoryRoleAssignments)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// AssignInventoryRole assigns one or more inventory roles to a user for a
// specific managed object.
//
// Per the OAS spec, "managedObject" in the request body is a plain string
// (the managed-object ID), for example:
//
//	map[string]any{
//	  "managedObject": "12345",
//	  "roles": []map[string]any{{"id": 1}},
//	}
func (s *Service) AssignInventoryRole(ctx context.Context, opt UserScopedOptions, body any) op.Result[jsonmodels.InventoryRoleAssignment] {
	return core.Execute(ctx, s.assignInventoryRoleB(opt, body), jsonmodels.NewInventoryRoleAssignment).IgnoreConflict()
}

func (s *Service) assignInventoryRoleB(opt UserScopedOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(ParamUserID, opt.UserID).
		SetBody(body).
		SetURL(ApiInventoryRoleAssignments)
	return core.NewTryRequest(s.Client, req)
}

// ---------------------------------------------------------------------------
// Inventory Role Assignment (single)
// ---------------------------------------------------------------------------

// GetInventoryRoleAssignment retrieves a single inventory-role assignment.
func (s *Service) GetInventoryRoleAssignment(ctx context.Context, opt GetInventoryRoleAssignmentOptions) op.Result[jsonmodels.InventoryRoleAssignment] {
	return core.Execute(ctx, s.getInventoryRoleAssignmentB(opt), jsonmodels.NewInventoryRoleAssignment)
}

func (s *Service) getInventoryRoleAssignmentB(opt GetInventoryRoleAssignmentOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(ParamUserID, opt.UserID).
		SetPathParam(ParamID, strconv.FormatInt(opt.ID, 10)).
		SetURL(ApiInventoryRoleAssignment)
	return core.NewTryRequest(s.Client, req)
}

// UpdateInventoryRoleAssignment modifies an existing inventory-role assignment.
// The body has the same shape as AssignInventoryRole.
func (s *Service) UpdateInventoryRoleAssignment(ctx context.Context, opt GetInventoryRoleAssignmentOptions, body any) op.Result[jsonmodels.InventoryRoleAssignment] {
	return core.Execute(ctx, s.updateInventoryRoleAssignmentB(opt, body), jsonmodels.NewInventoryRoleAssignment)
}

func (s *Service) updateInventoryRoleAssignmentB(opt GetInventoryRoleAssignmentOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(ParamUserID, opt.UserID).
		SetPathParam(ParamID, strconv.FormatInt(opt.ID, 10)).
		SetBody(body).
		SetURL(ApiInventoryRoleAssignment)
	return core.NewTryRequest(s.Client, req)
}

// DeleteInventoryRoleAssignment removes an inventory-role assignment.
func (s *Service) DeleteInventoryRoleAssignment(ctx context.Context, opt GetInventoryRoleAssignmentOptions) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteInventoryRoleAssignmentB(opt)).IgnoreNotFound()
}

func (s *Service) deleteInventoryRoleAssignmentB(opt GetInventoryRoleAssignmentOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(ParamUserID, opt.UserID).
		SetPathParam(ParamID, strconv.FormatInt(opt.ID, 10)).
		SetURL(ApiInventoryRoleAssignment)
	return core.NewTryRequest(s.Client, req)
}
