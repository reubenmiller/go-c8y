package jsonmodels

import (
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
)

// BulkOperationProgress holds the aggregated counts of child operations
// within a bulk operation.
type BulkOperationProgress struct {
	// Pending is the number of operations that have not yet started.
	Pending int64 `json:"pending"`

	// Failed is the number of operations that have failed.
	Failed int64 `json:"failed"`

	// Executing is the number of operations currently executing.
	Executing int64 `json:"executing"`

	// Successful is the number of operations that have completed successfully.
	Successful int64 `json:"successful"`

	// All is the total count of child operations.
	All int64 `json:"all"`
}

// BulkOperation wraps a Cumulocity bulk operation JSON document.
// It exposes typed accessors for the standard fields defined in the OAS spec.
type BulkOperation struct {
	jsondoc.Facade
}

// NewBulkOperation constructs a BulkOperation from a raw JSON byte slice.
func NewBulkOperation(b []byte) BulkOperation {
	return BulkOperation{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// ID returns the unique identifier of the bulk operation.
func (o BulkOperation) ID() string {
	return o.Get("id").String()
}

// Self returns the self link of the bulk operation.
func (o BulkOperation) Self() string {
	return o.Get("self").String()
}

// GroupID returns the ID of the device group targeted by this bulk operation.
func (o BulkOperation) GroupID() string {
	return o.Get("groupId").String()
}

// FailedParentID returns the ID of the previous bulk operation from which this
// one was created after a failure. Returns an empty string if not set.
func (o BulkOperation) FailedParentID() string {
	return o.Get("failedParentId").String()
}

// StartDate returns the scheduled start time of the bulk operation.
func (o BulkOperation) StartDate() time.Time {
	return o.Get("startDate").Time()
}

// CreationRamp returns the delay (in seconds) between the creation of
// successive individual child operations.
func (o BulkOperation) CreationRamp() float64 {
	return o.Get("creationRamp").Float()
}

// Status returns the requested status of the bulk operation
// (ACTIVE | DEFERRED | EXECUTING | CANCELED | COMPLETED).
func (o BulkOperation) Status() string {
	return o.Get("status").String()
}

// GeneralStatus returns the computed overall status of the bulk operation
// derived from the state of all its child operations
// (SCHEDULED | EXECUTING | EXECUTING_WITH_ERRORS | COMPLETED | DELETED | CANCELED).
func (o BulkOperation) GeneralStatus() string {
	return o.Get("generalStatus").String()
}

// Progress returns the aggregated progress counts of all child operations.
func (o BulkOperation) Progress() BulkOperationProgress {
	return BulkOperationProgress{
		Pending:    o.Get("progress.pending").Int(),
		Failed:     o.Get("progress.failed").Int(),
		Executing:  o.Get("progress.executing").Int(),
		Successful: o.Get("progress.successful").Int(),
		All:        o.Get("progress.all").Int(),
	}
}
