package jsonmodels

import "github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsondoc"

type SystemOption struct {
	jsondoc.JSONDoc
}

func NewSystemOption(b []byte) SystemOption {
	return SystemOption{jsondoc.New(b)}
}

// Category returns the category of the system option
func (o SystemOption) Category() string {
	return o.Get("category").String()
}

// Key returns the key of the system option
func (o SystemOption) Key() string {
	return o.Get("key").String()
}

// Value returns the value of the system option
func (o SystemOption) Value() string {
	return o.Get("value").String()
}
