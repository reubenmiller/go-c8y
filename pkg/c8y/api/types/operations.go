package types

type OperationStatus string

const (
	OperationStatusPending    OperationStatus = "PENDING"
	OperationStatusExecuting  OperationStatus = "EXECUTING"
	OperationStatusSuccessful OperationStatus = "SUCCESSFUL"
	OperationStatusFailed     OperationStatus = "FAILED"
)
