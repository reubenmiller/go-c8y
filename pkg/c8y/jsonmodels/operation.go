package jsonmodels

import (
	"encoding/json"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

type Operation struct {
	jsondoc.Facade
}

func NewOperation(b []byte) Operation {
	return Operation{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// NewOperationWithStatus returns an operation document for the device with the
// status, and the extra fragments (the operation's own fragment).
func NewOperationWithStatus(deviceID, status string, fragments map[string]any) Operation {
	data := map[string]any{
		"deviceId": deviceID,
		"status":   status,
	}
	for k, v := range fragments {
		data[k] = v
	}
	b, _ := json.Marshal(data)
	return Operation{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// BulkOperationID returns the reference to the bulk operation this operation was
// scheduled from. Hand-written so that it is a string like every other id (the
// OAS declares it as an integer).
func (m Operation) BulkOperationID() string { return m.Get("bulkOperationId").String() }

// DeviceName returns the operation's device name. Hand-written: `deviceName` is not a
// property in the OAS operation schema. The scalar accessors (ID, DeviceID, Status,
// CreationTime, ...) are generated in zz_generated_operation.go.
func (o Operation) DeviceName() string {
	return o.Get("deviceName").String()
}

// Description returns the operation's description. Hand-written: `description` is not a
// property in the OAS operation schema.
func (o Operation) Description() string {
	return o.Get("description").String()
}
