package operations

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
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
func (s *Service) List(ctx context.Context, opt ListOptions) (*model.OperationCollection, error) {
	return core.ExecuteResultOnly[model.OperationCollection](ctx, s.ListB(opt))
}

func (s *Service) ListB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiOperations)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get an operation
func (s *Service) Get(ctx context.Context, ID string) (*model.Operation, error) {
	return core.ExecuteResultOnly[model.Operation](ctx, s.GetB(ID))
}

func (s *Service) GetB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiOperation)
	return core.NewTryRequest(s.Client, req)
}

// Create an operation
func (s *Service) Create(ctx context.Context, body any) (*model.Operation, error) {
	return core.ExecuteResultOnly[model.Operation](ctx, s.CreateB(body))
}

func (s *Service) CreateB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetURL(ApiOperations)
	return core.NewTryRequest(s.Client, req)
}

// Update an operation
func (s *Service) Update(ctx context.Context, ID string, body any) (*model.Operation, error) {
	return core.ExecuteResultOnly[model.Operation](ctx, s.UpdateB(ID, body))
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
func (s *Service) DeleteList(ctx context.Context, opt DeleteListOptions) error {
	return core.ExecuteNoResult(ctx, s.DeleteListB(opt))
}

func (s *Service) DeleteListB(opt DeleteListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiOperations)
	return core.NewTryRequest(s.Client, req)
}
