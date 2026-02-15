package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
	"github.com/tidwall/gjson"
)

type UIPluginVersion struct {
	jsondoc.Facade
}

func NewUIPluginVersion(data []byte) UIPluginVersion {
	return UIPluginVersion{jsondoc.Facade{JSONDoc: jsondoc.New(data)}}
}

func (v *UIPluginVersion) Version() string {
	return v.Get("version").String()
}

func (v *UIPluginVersion) BinaryID() string {
	return v.Get("binaryId").String()
}

func (v *UIPluginVersion) Tags() []string {
	tags := []string{}
	v.Get("tags").ForEach(func(key, value gjson.Result) bool {
		tags = append(tags, value.String())
		return true
	})
	return tags
}
