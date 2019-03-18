package c8y

import (
	"context"

	"github.com/tidwall/gjson"
)

// OperationService todo
type OperationService service

// OperationCollectionOptions todo
type OperationCollectionOptions struct {
	// Source device to filter measurements by
	Status string `url:"status,omitempty"`

	AgentID string `url:"agentId,omitempty"`

	DeviceID string `url:"deviceId,omitempty"`

	// Pagination options
	PaginationOptions
}

// OperationCollection todo
type OperationCollection struct {
	*BaseResponse

	Operations []Operation `json:"operations"`

	Items []gjson.Result
}

// OperationStatus todo
type OperationStatus int

// Operation Status Contansts
const (
	Pending OperationStatus = iota
	Executing
	Failed
	Success
)

func (s OperationStatus) String() string {
	switch s {
	case Pending:
		return "PENDING"
	case Executing:
		return "EXECUTING"
	case Failed:
		return "FAILED"
	case Success:
		return "SUCCESSFUL"
	}
	return ""
}

// OperationUpdateOptions todo
type OperationUpdateOptions struct {
	// Status Operation status, can be one of SUCCESSFUL, FAILED, EXECUTING or PENDING
	Status string `json:"status,omitempty"`

	// FailureReason is the Reason for the failure
	FailureReason string `json:"failureReason,omitempty"`
}

// GetOperationCollection returns a collection of Cumulocity operations
func (s *OperationService) GetOperationCollection(ctx context.Context, opt *OperationCollectionOptions) (*OperationCollection, *Response, error) {
	data := new(OperationCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "devicecontrol/operations",
		Query:        opt,
		ResponseData: data,
	})
	data.Items = resp.JSON.Get("operations").Array()
	return data, resp, err
}

// DeleteCollection deletes a collection of Cumulocity operations
func (s *OperationService) DeleteCollection(ctx context.Context, opt *OperationCollectionOptions) (*Response, error) {
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "devicecontrol/operations",
		Query:  opt,
	})
	return resp, err
}

// CreateOperation creates a new operation for a device
func (s *OperationService) CreateOperation(ctx context.Context, body interface{}) (*Operation, *Response, error) {
	data := new(Operation)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "devicecontrol/operations",
		Body:         body,
		ResponseData: data,
	})
	data.Item = gjson.Parse(resp.JSON.Raw)
	return data, resp, err
}

// UpdateOperation updates a Cumulocity operation
func (s *OperationService) UpdateOperation(ctx context.Context, ID string, body *OperationUpdateOptions) (*Operation, *Response, error) {
	data := new(Operation)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "devicecontrol/operations/" + ID,
		Body:         body,
		ResponseData: data,
	})
	data.Item = gjson.Parse(resp.JSON.Raw)
	return data, resp, err
}
