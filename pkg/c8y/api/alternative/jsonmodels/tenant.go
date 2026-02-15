package jsonmodels

import (
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsondoc"
)

type Tenant struct {
	jsondoc.Facade
}

func NewTenant(b []byte) Tenant {
	return Tenant{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (t Tenant) ID() string {
	return t.Get("id").String()
}

func (t Tenant) Name() string {
	return t.Get("name").String()
}

func (t Tenant) Domain() string {
	return t.Get("domain").String()
}

func (t Tenant) Self() string {
	return t.Get("self").String()
}

func (t Tenant) CreationTime() time.Time {
	return t.Get("creationTime").Time()
}
