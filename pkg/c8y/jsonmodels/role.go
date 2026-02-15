package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
)

type Role struct {
	jsondoc.Facade
}

func NewRole(b []byte) Role {
	return Role{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (r Role) ID() string {
	return r.Get("id").String()
}

func (r Role) Name() string {
	return r.Get("name").String()
}

func (r Role) Self() string {
	return r.Get("self").String()
}
