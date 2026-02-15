package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsondoc"
)

type DeviceCredentials struct {
	jsondoc.Facade
}

func NewDeviceCredentials(b []byte) DeviceCredentials {
	return DeviceCredentials{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (i DeviceCredentials) ID() string {
	return i.Get("id").String()
}

func (i DeviceCredentials) TenantID() string {
	return i.Get("tenantId").String()
}

func (i DeviceCredentials) Username() string {
	return i.Get("username").String()
}

func (i DeviceCredentials) Password() string {
	return i.Get("password").String()
}

func (i DeviceCredentials) Self() string {
	return i.Get("self").String()
}
