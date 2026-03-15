package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

type ApplicationReference struct {
	jsondoc.Facade
}

func NewApplicationReference(b []byte) ApplicationReference {
	return ApplicationReference{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (a ApplicationReference) Self() string {
	return a.Get("self").String()
}

func (a ApplicationReference) Application() Application {
	return NewApplication([]byte(a.Get("application").Raw))
}
