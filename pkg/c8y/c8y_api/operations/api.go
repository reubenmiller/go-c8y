package operations

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
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

// List operations
func (s *Service) List(ctx context.Context, opt ListOptions) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiOperations)
}

func (s *Service) ListPager(ctx context.Context, opt ListOptions) *core.TryRequest {
	return &core.TryRequest{
		Client:   s.Client,
		Request:  s.List(ctx, opt),
		Property: ResultProperty,
	}
}

// Get an operation
func (s *Service) Get(ctx context.Context, ID string) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiOperation)
}

// Create an operation
func (s *Service) Create(ctx context.Context, body any) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetURL(ApiOperations)
}

// Update an operation
func (s *Service) Update(ctx context.Context, ID string, body any) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetBody(body).
		SetURL(ApiOperations)
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
func (s *Service) DeleteList(ctx context.Context, opt DeleteListOptions) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiOperations)
}
