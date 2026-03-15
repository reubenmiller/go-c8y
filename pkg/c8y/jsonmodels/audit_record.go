package jsonmodels

import (
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

type AuditRecord struct {
	jsondoc.Facade
}

func NewAuditRecord(b []byte) AuditRecord {
	return AuditRecord{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (a AuditRecord) ID() string {
	return a.Get("id").String()
}

func (a AuditRecord) Type() string {
	return a.Get("type").String()
}

func (a AuditRecord) Activity() string {
	return a.Get("activity").String()
}

func (a AuditRecord) User() string {
	return a.Get("user").String()
}

func (a AuditRecord) Application() string {
	return a.Get("application").String()
}

func (a AuditRecord) Time() time.Time {
	return a.Get("time").Time()
}

func (a AuditRecord) CreationTime() time.Time {
	return a.Get("creationTime").Time()
}

func (a AuditRecord) Self() string {
	return a.Get("self").String()
}
