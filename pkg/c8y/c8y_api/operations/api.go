package operations

import (
	"context"
	"iter"
	"log/slog"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiOperations = "/devicecontrol/operations"
var ApiOperation = "/devicecontrol/operations/{id}"

var ParamId = "id"

const ResultProperty = "operations"

// Service provides api to get/set/delete operations
type Service core.Service

// ListOptions to use when search for operations
type ListOptions struct {
	// An agent ID that may be part of the operation. If this parameter is set,
	// the operation response objects contain the deviceExternalIDs object.
	AgentID string `url:"agentId,omitempty"`

	// The bulk operation ID that this operation belongs to
	BulkOperationID string `url:"bulkOperationId,omitempty"`

	// Start date or date and time of the operation
	DateFrom time.Time `url:"dateFrom,omitempty,omitzero"`

	// End date or date and time of the operation
	DateTo time.Time `url:"dateTo,omitempty,omitzero"`

	// The ID of the device the operation is performed for
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
type OperationIterator struct {
	items iter.Seq[jsonmodels.Operation]
	err   error
}

func (it *OperationIterator) Items() iter.Seq[jsonmodels.Operation] {
	return it.items
}

func (it *OperationIterator) Err() error {
	return it.err
}

func paginateOperations(ctx context.Context, fetch func(page int) op.Result[jsonmodels.Operation], maxItems int) *OperationIterator {
	iterator := &OperationIterator{}

	iterator.items = func(yield func(jsonmodels.Operation) bool) {
		page := 1
		count := 0
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
				item := jsonmodels.NewOperation(doc.Bytes())
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

// List operations
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Operation] {
	return core.ExecuteReturnCollection(ctx, s.ListB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewOperation)
}

// ListAll returns an iterator for all operations
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *OperationIterator {
	return paginateOperations(ctx, func(page int) op.Result[jsonmodels.Operation] {
		opts.CurrentPage = page
		opts.PageSize = 2000
		return s.List(ctx, opts)
	}, 0)
}

// ListLimit returns an iterator for up to maxItems operations
func (s *Service) ListLimit(ctx context.Context, opts ListOptions, maxItems int) *OperationIterator {
	return paginateOperations(ctx, func(page int) op.Result[jsonmodels.Operation] {
		opts.CurrentPage = page
		opts.PageSize = 2000
		return s.List(ctx, opts)
	}, maxItems)
}

func (s *Service) ListB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiOperations)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get an operation
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.Operation] {
	return core.ExecuteReturnResult(ctx, s.GetB(ID), jsonmodels.NewOperation)
}

func (s *Service) GetB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiOperation)
	return core.NewTryRequest(s.Client, req)
}

// Create an operation
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Operation] {
	return core.ExecuteReturnResult(ctx, s.CreateB(body), jsonmodels.NewOperation)
}

func (s *Service) CreateB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetURL(ApiOperations)
	return core.NewTryRequest(s.Client, req)
}

// Update an operation
func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.Operation] {
	return core.ExecuteReturnResult(ctx, s.UpdateB(ID, body), jsonmodels.NewOperation)
}

func (s *Service) UpdateB(ID string, body any) *core.TryRequest {
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
	return core.ExecuteReturnResult(ctx, s.DeleteListB(opt), jsonmodels.NewOperation)
}

func (s *Service) DeleteListB(opt DeleteListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiOperations)
	return core.NewTryRequest(s.Client, req)
}
