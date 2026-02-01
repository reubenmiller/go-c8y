package jsonmodels

import (
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
)

type Binary struct {
	jsondoc.Facade
}

func NewBinary(b []byte) Binary {
	return Binary{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (b Binary) ID() string {
	return b.Get("id").String()
}

func (b Binary) Name() string {
	return b.Get("name").String()
}

func (b Binary) Type() string {
	return b.Get("type").String()
}

func (b Binary) Self() string {
	return b.Get("self").String()
}

func (b Binary) CreationTime() time.Time {
	return b.Get("creationTime").Time()
}

func (b Binary) LastUpdated() time.Time {
	return b.Get("lastUpdated").Time()
}

func (b Binary) Length() int64 {
	return b.Get("length").Int()
}
