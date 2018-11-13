package c8y

import (
	"context"
	"fmt"
	"log"
)

// OperationService todo
type OperationService service

/* const (
	// Pending todo
	Pending = "PENDING"

	// Executing todo
	Executing = "EXECUTING"

	// Successful todo
	Successful = "SUCCESSFUL"

	// Failed todo
	Failed = "FAILED"
) */

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
	*C8yBaseResponse

	Operations []Operation `json:"operations"`
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

// GetOperationCollection todo
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

	opCol := new(OperationCollection)

	resp, err := s.client.Do(ctx, req, opCol)
	if err != nil {
		return nil, resp, err
	}

	if opt.PaginationOptions.WithTotalPages == true {
		log.Printf("Total operations: %d\n", *opCol.Statistics.TotalPages)
	} else {
		log.Printf("Total operations: %d\n", len(opCol.Operations))
	}

	return opCol, resp, nil
}

// UpdateOperation todo
func (s *OperationService) UpdateOperation(ctx context.Context, ID string, body *OperationUpdateOptions) (*Operation, *Response, error) {
	u := fmt.Sprintf("devicecontrol/operations/%s", ID)

	req, err := s.client.NewRequest("PUT", u, "", body)
	if err != nil {
		return nil, nil, err
	}

	opCol := new(Operation)

	resp, err := s.client.Do(ctx, req, opCol)
	if err != nil {
		return nil, resp, err
	}

	return opCol, resp, nil
}
