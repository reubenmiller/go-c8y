package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
)

type Microservice struct {
	jsondoc.Facade
}

func NewMicroservice(b []byte) Microservice {
	return Microservice{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (m Microservice) ID() string {
	return m.Get("id").String()
}

func (m Microservice) Name() string {
	return m.Get("name").String()
}

func (m Microservice) Key() string {
	return m.Get("key").String()
}

func (m Microservice) Type() string {
	return m.Get("type").String()
}

func (m Microservice) Manifest() jsondoc.JSONDoc {
	return jsondoc.New([]byte(m.Get("manifest").Raw))
}

func (m Microservice) Self() string {
	return m.Get("self").String()
}
