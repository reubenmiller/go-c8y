package jsonmodels

import (
	"encoding/json"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
)

type Operation struct {
	jsondoc.Facade
}

func NewOperation(b []byte) Operation {
	return Operation{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

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

func (o Operation) ID() string {
	return o.Get("id").String()
}

func (o Operation) DeviceID() string {
	return o.Get("deviceId").String()
}

func (o Operation) DeviceName() string {
	return o.Get("deviceName").String()
}

func (o Operation) Status() string {
	return o.Get("status").String()
}

func (o Operation) Description() string {
	return o.Get("description").String()
}

func (o Operation) CreationTime() time.Time {
	return o.Get("creationTime").Time()
}

func (o Operation) Self() string {
	return o.Get("self").String()
}
