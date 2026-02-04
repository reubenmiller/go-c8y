package operations

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/internal/context"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiOperations = "/devicecontrol/operations"
var ApiOperation = "/devicecontrol/operations/{id}"

var ParamId = "id"

const ResultProperty = "operations"

// Service provides api to get/set/delete operations
type Service struct {
	core.Service
	DeviceResolver *managedobjects.DeviceResolver
}

// NewService creates a new operations service with device resolution capabilities
func NewService(common *core.Service, moService *managedobjects.Service) *Service {
	return &Service{
		Service:        *common,
		DeviceResolver: managedobjects.NewDeviceResolver(moService),
	}
}

// ListOptions to use when search for operations
type ListOptions struct {
	// An agent ID that may be part of the operation. If this parameter is set,
	// the operation response objects contain the deviceExternalIDs object.
	// Supports resolver strings: direct ID, "name:deviceName", "ext:type:id", "query:..."
	AgentID string `url:"agentId,omitempty"`

	// The bulk operation ID that this operation belongs to
	BulkOperationID string `url:"bulkOperationId,omitempty"`

	// Start date or date and time of the operation
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the operation
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// The ID of the device the operation is performed for.
	// Supports resolver strings: direct ID, "name:deviceName", "ext:type:id", "query:..."
	DeviceID string `url:"deviceId,omitempty"`

	// The type of fragment that must be part of the operation
	FragmentType string `url:"fragmentType,omitempty"`

	// If you are using a range query (that is, at least one of
	// the dateFrom or dateTo parameters is included in the request),
	// then setting revert=true will sort the results by the newest operations
	// first. By default, the results are sorted by the oldest operations first.
	Revert bool `url:"revert,omitempty"`

	// Status of the operation
	Status string `url:"status,omitempty"`

	pagination.PaginationOptions
}

// OperationIterator provides iteration over operations
type OperationIterator = pagination.Iterator[jsonmodels.Operation]

// List operations
// The DeviceID and AgentID fields support resolver strings:
//   - "12345" - direct ID
//   - "name:deviceName" - lookup by device name
//   - "ext:c8y_Serial:ABC123" - lookup by external ID
//   - "query:type eq 'c8y_Device'" - lookup by inventory query
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Operation] {
	// Resolve DeviceID if it contains a resolver scheme
	if opt.DeviceID != "" && s.DeviceResolver != nil {
		resolutionCtx := ctx
		if ctxhelpers.IsDeferredExecution(ctx) {
			resolutionCtx = context.Background()
		}

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.DeviceID, nil)
		if err != nil {
			return op.Failed[jsonmodels.Operation](err, true)
		}
		opt.DeviceID = resolvedID
	}

	// Resolve AgentID if it contains a resolver scheme
	if opt.AgentID != "" && s.DeviceResolver != nil {
		resolutionCtx := ctx
		if ctxhelpers.IsDeferredExecution(ctx) {
			resolutionCtx = context.Background()
		}

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.AgentID, nil)
		if err != nil {
			return op.Failed[jsonmodels.Operation](err, true)
		}
		opt.AgentID = resolvedID
	}

	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewOperation)
}

// ListAll returns an iterator for all operations
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *OperationIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Operation] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewOperation,
	)
}

func (s *Service) listB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiOperations)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get an operation
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.Operation] {
	return core.Execute(ctx, s.getB(ID), jsonmodels.NewOperation)
}

func (s *Service) getB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiOperation)
	return core.NewTryRequest(s.Client, req)
}

// Create an operation
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Operation] {
	return core.Execute(ctx, s.createB(body), jsonmodels.NewOperation)
}

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetURL(ApiOperations)
	return core.NewTryRequest(s.Client, req)
}

// Update an operation
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.Operation] {
	return core.Execute(ctx, s.updateB(ID, body), jsonmodels.NewOperation)
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetBody(body).
		SetURL(ApiOperation)
	return core.NewTryRequest(s.Client, req)
}

// Delete a list of operations
type DeleteListOptions struct {
	// An agent ID that may be part of the operation
	AgentID string `url:"agentId,omitempty"`

	// Start date or date and time of the operation.
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the operation
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// The ID of the device the operation is performed for
	DeviceID string `url:"deviceId,omitempty"`

	// Status of the operation
	Status string `url:"status,omitempty"`
}

// Delete a list of operations
func (s *Service) DeleteList(ctx context.Context, opt DeleteListOptions) op.Result[jsonmodels.Operation] {
	// Resolve DeviceID if it contains a resolver scheme
	if opt.DeviceID != "" && s.DeviceResolver != nil {
		resolutionCtx := ctx
		if ctxhelpers.IsDeferredExecution(ctx) {
			resolutionCtx = context.Background()
		}

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.DeviceID, nil)
		if err != nil {
			return op.Failed[jsonmodels.Operation](err, true)
		}
		opt.DeviceID = resolvedID
	}

	// Resolve AgentID if it contains a resolver scheme
	if opt.AgentID != "" && s.DeviceResolver != nil {
		resolutionCtx := ctx
		if ctxhelpers.IsDeferredExecution(ctx) {
			resolutionCtx = context.Background()
		}

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.AgentID, nil)
		if err != nil {
			return op.Failed[jsonmodels.Operation](err, true)
		}
		opt.AgentID = resolvedID
	}

	return core.Execute(ctx, s.deleteListB(opt), jsonmodels.NewOperation)
}

func (s *Service) deleteListB(opt DeleteListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiOperations)
	return core.NewTryRequest(s.Client, req)
}
