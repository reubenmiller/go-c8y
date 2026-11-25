// Package types covers shared enums and small interfaces used by the service
// packages: operation and bulk operation statuses, processing mode, media
// types.
//
// It is used by the other packages and rarely imported directly.
//
// Example, with a client and ctx in scope:
//
//	fmt.Println(types.OperationStatusPending, types.ProcessingModeTransient)
package types
