package core

import (
	"fmt"
	"net/http"
)

// BatchApplied says whether a failed batch request wrote anything.
type BatchApplied int

const (
	// BatchAppliedNone means the platform validated the request before writing
	// and nothing was written.
	BatchAppliedNone BatchApplied = iota

	// BatchAppliedPartial means some items may have been written. Marker
	// identifies them.
	BatchAppliedPartial

	// BatchAppliedUnknown means the platform gave no information about what was
	// written. Check the affected resources before retrying.
	BatchAppliedUnknown
)

func (a BatchApplied) String() string {
	switch a {
	case BatchAppliedNone:
		return "none"
	case BatchAppliedPartial:
		return "partial"
	default:
		return "unknown"
	}
}

// BulkRequestInfo is the "bulkRequest" object that Cumulocity adds to the error
// of a request carrying several items.
type BulkRequestInfo struct {
	// FailedIndex is the position, counted from 0, of the item the error belongs to.
	FailedIndex *int `json:"failedIndex,omitempty"`

	// FragmentType is the marker fragment carried by items that were written
	// before the failure. Present only when part of the request was written.
	FragmentType string `json:"fragmentType,omitempty"`
}

// BatchError is returned by any request that carries several items (for
// example a bulk create of managed objects, or creating several measurements)
// when the platform reports which item failed. Get it with errors.As:
//
//	var be *core.BatchError
//	if errors.As(result.Err, &be) && be.Applied == core.BatchAppliedPartial {
//		// part of the request was written; see be.Marker
//	}
//
// It wraps the HTTP error, so errors.As with *core.Error also works.
type BatchError struct {
	// Err is the HTTP error (status, error type, message).
	Err *Error

	// Index is the position of the failing item in the request, counted from 0.
	// It is -1 when the error is about the whole request.
	Index int

	// Offset is the position of the request's first item in a larger input,
	// set by streaming helpers that split input into several requests. The
	// position of the failing item in the whole input is Offset + Index.
	Offset int

	// Applied says whether anything was written before the failure.
	Applied BatchApplied

	// Marker identifies the items that were written, so they can be found
	// again. For managed objects it is the c8y_BatchId_* fragment type.
	Marker string
}

// Error implements the error interface.
func (e *BatchError) Error() string {
	msg := ""
	if e.Err != nil {
		msg = e.Err.Error()
	}
	if e.Index >= 0 {
		return fmt.Sprintf("batch request failed at item %d (written: %s): %s", e.Offset+e.Index, e.Applied, msg)
	}
	return fmt.Sprintf("batch request failed (written: %s): %s", e.Applied, msg)
}

// Unwrap returns the wrapped HTTP error.
func (e *BatchError) Unwrap() error {
	if e.Err == nil {
		return nil
	}
	return e.Err
}

// StatusCode returns the HTTP status code of the failed request.
func (e *BatchError) StatusCode() int {
	if e.Err == nil {
		return 0
	}
	return e.Err.StatusCode()
}

// NewBatchError builds a BatchError from an HTTP error and the bulkRequest
// information in its body. It returns nil when info is nil.
func NewBatchError(err *Error, info *BulkRequestInfo) *BatchError {
	if info == nil {
		return nil
	}
	be := &BatchError{
		Err:    err,
		Index:  -1,
		Marker: info.FragmentType,
	}
	if info.FailedIndex != nil {
		be.Index = *info.FailedIndex
	}
	status := 0
	if err != nil {
		status = err.StatusCode()
	}
	switch {
	case be.Marker != "":
		be.Applied = BatchAppliedPartial
	case status == http.StatusNotFound, status == http.StatusConflict, status == http.StatusUnprocessableEntity:
		be.Applied = BatchAppliedNone
	default:
		be.Applied = BatchAppliedUnknown
	}
	return be
}
