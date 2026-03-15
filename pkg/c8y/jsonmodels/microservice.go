package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
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

func (m Microservice) Owner() string {
	return m.Get("owner.tenant.id").String()
}

func (m Microservice) Availability() string {
	return m.Get("availability").String()
}

func (m Microservice) RequiredRoles() []string {
	node := m.Get("requiredRoles").Array()
	items := make([]string, 0, len(node))
	for _, item := range node {
		items = append(items, item.String())
	}
	return items
}

func (m Microservice) Roles() []string {
	node := m.Get("roles").Array()
	items := make([]string, 0, len(node))
	for _, item := range node {
		items = append(items, item.String())
	}
	return items
}

func (m Microservice) ContextPath() string {
	return m.Get("contextPath").String()
}

func (m Microservice) Manifest() jsondoc.JSONDoc {
	return jsondoc.New([]byte(m.Get("manifest").Raw))
}

func (m Microservice) Self() string {
	return m.Get("self").String()
}
