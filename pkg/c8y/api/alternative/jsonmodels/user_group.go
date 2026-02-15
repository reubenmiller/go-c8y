package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsondoc"
)

type UserGroup struct {
	jsondoc.Facade
}

func NewUserGroup(b []byte) UserGroup {
	return UserGroup{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (g UserGroup) ID() string {
	return g.Get("id").String()
}

func (g UserGroup) Name() string {
	return g.Get("name").String()
}

func (g UserGroup) Self() string {
	return g.Get("self").String()
}
