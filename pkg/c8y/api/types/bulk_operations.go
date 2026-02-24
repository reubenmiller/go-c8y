package types

// BulkOperationStatus is the requested status of a bulk operation.
type BulkOperationStatus string

const (
	// BulkOperationActive marks the bulk operation as active (not yet started).
	BulkOperationActive BulkOperationStatus = "ACTIVE"

	// BulkOperationInProgress means the bulk operation is currently being processed.
	BulkOperationInProgress BulkOperationStatus = "IN_PROGRESS"

	// BulkOperationCompleted means all individual operations have been completed.
	BulkOperationCompleted BulkOperationStatus = "COMPLETED"

	// BulkOperationDeleted means the bulk operation has been deleted.
	BulkOperationDeleted BulkOperationStatus = "DELETED"
)

// BulkOperationGeneralStatus is the overall/aggregated status of a bulk operation
// derived from the state of all its individual child operations.
type BulkOperationGeneralStatus string

const (
	// BulkOperationGeneralScheduled means the bulk operation is scheduled but not yet started.
	BulkOperationGeneralScheduled BulkOperationGeneralStatus = "SCHEDULED"

	// BulkOperationGeneralExecuting means some child operations are currently running.
	BulkOperationGeneralExecuting BulkOperationGeneralStatus = "EXECUTING"

	// BulkOperationGeneralExecutingWithErrors means execution is ongoing but some operations have failed.
	BulkOperationGeneralExecutingWithErrors BulkOperationGeneralStatus = "EXECUTING_WITH_ERRORS"

	// BulkOperationGeneralSuccessful means all child operations completed successfully.
	BulkOperationGeneralSuccessful BulkOperationGeneralStatus = "SUCCESSFUL"

	// BulkOperationGeneralFailed means all child operations have finished and at least one failed.
	BulkOperationGeneralFailed BulkOperationGeneralStatus = "FAILED"

	// BulkOperationGeneralCanceled means the bulk operation was canceled.
	BulkOperationGeneralCanceled BulkOperationGeneralStatus = "CANCELED"
)
