package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// DatabrokerConnector is a Cumulocity data-broker connector.
type DatabrokerConnector struct {
	jsondoc.Facade
}

func NewDatabrokerConnector(b []byte) DatabrokerConnector {
	return DatabrokerConnector{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (c DatabrokerConnector) ID() string {
	return c.Get("id").String()
}

func (c DatabrokerConnector) Name() string {
	return c.Get("name").String()
}

func (c DatabrokerConnector) Status() string {
	return c.Get("status").String()
}
