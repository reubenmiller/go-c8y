package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
)

type RetentionRule struct {
	jsondoc.Facade
}

func NewRetentionRule(b []byte) RetentionRule {
	return RetentionRule{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (r RetentionRule) ID() string {
	return r.Get("id").String()
}

func (r RetentionRule) DataType() string {
	return r.Get("dataType").String()
}

func (r RetentionRule) FragmentType() string {
	return r.Get("fragmentType").String()
}

func (r RetentionRule) MaximumAge() int64 {
	return r.Get("maximumAge").Int()
}

func (r RetentionRule) Editable() bool {
	return r.Get("editable").Bool()
}

func (r RetentionRule) Self() string {
	return r.Get("self").String()
}
