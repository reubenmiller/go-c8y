package jsonmodels

import (
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
)

type ManagedObject struct {
	jsondoc.Facade
}

func NewManagedObject(b []byte) ManagedObject {
	return ManagedObject{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (m ManagedObject) ID() string {
	return m.Get("id").String()
}

func (m ManagedObject) Type() string {
	return m.Get("type").String()
}

func (m ManagedObject) Owner() string {
	return m.Get("owner").String()
}

func (m ManagedObject) CreationTime() time.Time {
	return m.Get("creationTime").Time()
}

func (m ManagedObject) LastUpdated() time.Time {
	return m.Get("lastUpdated").Time()
}

func (m ManagedObject) Name() string {
	return m.Get("name").String()
}

func (m ManagedObject) WithName(name string) (ManagedObject, error) {
	doc, err := m.Set("name", name)
	return ManagedObject{jsondoc.Facade{JSONDoc: doc}}, err
}
