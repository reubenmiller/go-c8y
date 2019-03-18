package c8y

import (
	"context"
	"fmt"

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
	Status string `json:"status"`
}

// GetOperationCollection returns a collection of Cumulocity operations
func (s *OperationService) GetOperationCollection(ctx context.Context, opt *OperationCollectionOptions) (*OperationCollection, *Response, error) {
	u := fmt.Sprintf("devicecontrol/operations")

	queryParams, err := addOptions("", opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, queryParams, nil)
	if err != nil {
		return nil, nil, err
	}

	data := new(OperationCollection)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	data.Items = resp.JSON.Get("operations").Array()

	return data, resp, nil
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
	u := fmt.Sprintf("devicecontrol/operations/%s", ID)

	req, err := s.client.NewRequest("PUT", u, "", body)
	if err != nil {
		return nil, nil, err
	}

	data := new(Operation)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	data.Item = *resp.JSON

	return data, resp, nil
}
