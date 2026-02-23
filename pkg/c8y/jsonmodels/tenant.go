package jsonmodels

import (
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
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

type CurrentTenant struct {
	jsondoc.Facade
}

func NewCurrentTenant(b []byte) CurrentTenant {
	return CurrentTenant{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (t CurrentTenant) ID() string {
	return t.Get("id").String()
}

func (t CurrentTenant) Name() string {
	return t.Get("name").String()
}

func (t CurrentTenant) DomainName() string {
	return t.Get("domainName").String()
}

func (t CurrentTenant) Self() string {
	return t.Get("self").String()
}

func (t CurrentTenant) CreationTime() time.Time {
	return t.Get("creationTime").Time()
}
