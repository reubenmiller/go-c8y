package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

type DeviceRequest struct {
	jsondoc.Facade
}

func NewDeviceRequest(b []byte) DeviceRequest {
	return DeviceRequest{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (i DeviceRequest) ID() string {
	return i.Get("id").String()
}

func (i DeviceRequest) Status() string {
	return i.Get("status").String()
}

func (i DeviceRequest) Self() string {
	return i.Get("self").String()
}
