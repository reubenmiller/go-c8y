package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

type Identity struct {
	jsondoc.Facade
}

func NewIdentity(b []byte) Identity {
	return Identity{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (i Identity) ExternalID() string {
	return i.Get("externalId").String()
}

func (i Identity) Type() string {
	return i.Get("type").String()
}

func (i Identity) Self() string {
	return i.Get("self").String()
}

func (i Identity) ManagedObjectID() string {
	return i.Get("managedObject.id").String()
}

func (i Identity) ManagedObjectSelf() string {
	return i.Get("managedObject.self").String()
}
