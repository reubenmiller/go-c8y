package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsondoc"
)

type Feature struct {
	jsondoc.Facade
}

func NewFeature(b []byte) Feature {
	return Feature{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (f Feature) Key() string {
	return f.Get("key").String()
}

func (f Feature) Phase() string {
	return f.Get("phase").String()
}

func (f Feature) Active() bool {
	return f.Get("active").Bool()
}

func (f Feature) Strategy() string {
	return f.Get("strategy").String()
}

func (f Feature) TenantId() string {
	return f.Get("tenantId").String()
}
