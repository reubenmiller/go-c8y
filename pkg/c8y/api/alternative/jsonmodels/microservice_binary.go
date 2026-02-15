package jsonmodels

import "github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsondoc"

type MicroserviceBinary struct {
	jsondoc.JSONDoc
}

func NewMicroserviceBinary(b []byte) MicroserviceBinary {
	return MicroserviceBinary{jsondoc.New(b)}
}

// ID returns the unique identifier of the microservice binary
func (m MicroserviceBinary) ID() string {
	return m.Get("id").String()
}

// Name returns the name of the microservice binary
func (m MicroserviceBinary) Name() string {
	return m.Get("name").String()
}

// Self returns the URL to this resource
func (m MicroserviceBinary) Self() string {
	return m.Get("self").String()
}

// Type returns the type of the managed object
func (m MicroserviceBinary) Type() string {
	return m.Get("type").String()
}
