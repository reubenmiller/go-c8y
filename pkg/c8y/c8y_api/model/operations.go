package model

import (
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
)

type Operation struct {
	ID            string                `json:"id,omitempty"`
	CreationTime  time.Time             `json:"creationTime,omitempty,omitzero"`
	DeviceID      string                `json:"deviceId,omitempty"`
	DeviceName    string                `json:"deviceName,omitempty"`
	Status        types.OperationStatus `json:"status,omitempty"`
	Description   string                `json:"description,omitempty"`
	Self          string                `json:"self,omitempty"`
	EventID       string                `json:"eventId,omitempty"`
	FailureReason string                `json:"failureReason,omitempty"`
}

// OperationCollection collection of alarms
type OperationCollection struct {
	*BaseResponse

	Operations []Operation `json:"operations"`
}
