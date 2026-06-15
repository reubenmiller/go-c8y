package jsonmodels

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"

// ApplicationBinary represents a binary attachment of an application.
type ApplicationBinary struct {
	jsondoc.Facade
}

func NewApplicationBinary(data []byte) ApplicationBinary {
	return ApplicationBinary{jsondoc.Facade{JSONDoc: jsondoc.New(data)}}
}

func (b ApplicationBinary) ID() string {
	return b.Get("id").String()
}

func (b ApplicationBinary) Name() string {
	return b.Get("name").String()
}

func (b ApplicationBinary) Version() string {
	return b.Get("version").String()
}
