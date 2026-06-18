// Package bulkoperations provides a client service for the Cumulocity
// Bulk Operations API (/devicecontrol/bulkoperations).
//
// A bulk operation targets a group of devices with a single operation
// template. The platform fans out individual child operations to every
// device in the group according to the configured schedule and ramp.
//
// Required roles:
//   - ROLE_BULK_OPERATION_READ  - for read access (List, Get)
//   - ROLE_BULK_OPERATION_ADMIN - for write access (Create, Update, Delete)
package bulkoperations

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// ApiBulkOperations is the collection endpoint for bulk operations.
var ApiBulkOperations = "/devicecontrol/bulkoperations"

// ApiBulkOperation is the single-item endpoint for bulk operations.
var ApiBulkOperation = "/devicecontrol/bulkoperations/{id}"

// ParamID is the path-parameter name used in ApiBulkOperation.
var ParamID = "id"

// ResultProperty is the JSON key that wraps the array of bulk operations
// in a collection response.
const ResultProperty = "bulkOperations"

// NewService creates a new BulkOperations service backed by the provided
// core.Service (HTTP client + tenant configuration).
func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service provides access to the Cumulocity Bulk Operations API.
type Service struct {
	core.Service
}

// ListOptions controls filtering and pagination of a bulk-operation list request.
type ListOptions struct {
	// WithDeleted includes CANCELLED bulk operations in the results.
	WithDeleted bool `url:"withDeleted,omitempty"`

	// DateFrom is the start date (or date and time) of the bulk operation.
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// DateTo is the end date (or date and time) of the bulk operation.
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// GeneralStatus filters by the general status of the bulk operation
	// (e.g. SCHEDULED, EXECUTING, FAILED). Multiple values are OR-combined.
	GeneralStatus []string `url:"generalStatus,omitempty"`

	pagination.PaginationOptions
}

// BulkOperationIterator is a lazy iterator over a (potentially multi-page)
// collection of bulk operations.
type BulkOperationIterator = pagination.Iterator[jsonmodels.BulkOperation]

// List returns a single page of bulk operations.
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.BulkOperation] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewBulkOperation)
}

// ListAll returns a lazy iterator that transparently pages through all
// bulk operations matching the given options.
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *BulkOperationIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.BulkOperation] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewBulkOperation,
	)
}

func (s *Service) listB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiBulkOperations)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get retrieves a single bulk operation by its ID.
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.BulkOperation] {
	return core.Execute(ctx, s.getB(ID), jsonmodels.NewBulkOperation)
}

func (s *Service) getB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, ID).
		SetURL(ApiBulkOperation)
	return core.NewTryRequest(s.Client, req)
}

// Create submits a new bulk operation. The body must contain at minimum
// groupId, startDate, creationRamp and operationPrototype.
//
// Any serialisable value (struct, map[string]any, etc.) is accepted.
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.BulkOperation] {
	return core.Execute(ctx, s.createB(body), jsonmodels.NewBulkOperation)
}

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiBulkOperations)
	return core.NewTryRequest(s.Client, req)
}

// Update modifies an existing bulk operation. Updatable fields include
// status, startDate, creationRamp and operationPrototype.
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.BulkOperation] {
	return core.Execute(ctx, s.updateB(ID, body), jsonmodels.NewBulkOperation)
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, ID).
		SetBody(body).
		SetURL(ApiBulkOperation)
	return core.NewTryRequest(s.Client, req)
}

// Delete removes a bulk operation by its ID. A 204 No Content response
// indicates success.
func (s *Service) Delete(ctx context.Context, ID string) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(ID)).IgnoreNotFound()
}

func (s *Service) deleteB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamID, ID).
		SetURL(ApiBulkOperation)
	return core.NewTryRequest(s.Client, req)
}

// --- Operations belonging to a bulk operation -------------------------------
//
// A bulk operation fans out into many individual device operations, served by
// the operations collection (/devicecontrol/operations) filtered by the
// bulkOperationId query parameter. The helpers below list those child
// operations straight from the BulkOperations service, so callers don't have to
// reach for the operations service and remember the filter name.

// ApiOperations is the operations collection endpoint, used to list the
// individual operations spawned by a bulk operation.
var ApiOperations = "/devicecontrol/operations"

// OperationsResultProperty is the JSON key that wraps the array of operations
// in a collection response.
const OperationsResultProperty = "operations"

// OperationIterator is a lazy iterator over a (potentially multi-page)
// collection of operations belonging to a bulk operation.
type OperationIterator = pagination.Iterator[jsonmodels.Operation]

// ListOperationsOptions controls filtering and pagination of the operations
// that belong to a single bulk operation.
type ListOperationsOptions struct {
	// BulkOperationID restricts the results to operations spawned by this bulk
	// operation. Without it the operations collection is returned unfiltered.
	BulkOperationID string `url:"bulkOperationId,omitempty"`

	// DateFrom is the start date (or date and time) of the operation.
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero" layout:"2006-01-02T15:04:05.000Z07:00"`

	// DateTo is the end date (or date and time) of the operation.
	DateTo time.Time `url:"dateTo,omitempty,omitzero" layout:"2006-01-02T15:04:05.000Z07:00"`

	// Status filters by operation status: one of PENDING, EXECUTING,
	// SUCCESSFUL or FAILED.
	Status string `url:"status,omitempty"`

	// Revert sorts operations newest-to-oldest. Only meaningful together with
	// DateFrom and/or DateTo.
	Revert bool `url:"revert,omitempty"`

	pagination.PaginationOptions
}

// ListOperations returns a single page of operations belonging to a bulk
// operation. Set BulkOperationID on opt to scope the results.
func (s *Service) ListOperations(ctx context.Context, opt ListOperationsOptions) op.Result[jsonmodels.Operation] {
	return core.ExecuteCollection(ctx, s.listOperationsB(opt), OperationsResultProperty, types.ResponseFieldStatistics, jsonmodels.NewOperation)
}

// ListAllOperations returns a lazy iterator that transparently pages through
// all operations belonging to a bulk operation matching the given options.
func (s *Service) ListAllOperations(ctx context.Context, opts ListOperationsOptions) *OperationIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Operation] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.ListOperations(ctx, o)
		},
		jsonmodels.NewOperation,
	)
}

func (s *Service) listOperationsB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiOperations)
	return core.NewTryRequest(s.Client, req, OperationsResultProperty)
}
