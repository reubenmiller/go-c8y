package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsondoc"
)

type UIPlugin struct {
	jsondoc.Facade
}

func NewUIPlugin(data []byte) UIPlugin {
	return UIPlugin{jsondoc.Facade{JSONDoc: jsondoc.New(data)}}
}

func (a *UIPlugin) ID() string {
	return a.Get("id").String()
}

func (a *UIPlugin) Self() string {
	return a.Get("self").String()
}

func (a *UIPlugin) Name() string {
	return a.Get("name").String()
}

func (a *UIPlugin) Key() string {
	return a.Get("key").String()
}

func (a *UIPlugin) Type() string {
	return a.Get("type").String()
}

func (a *UIPlugin) ContextPath() string {
	return a.Get("contextPath").String()
}

func (a *UIPlugin) Availability() string {
	return a.Get("availability").String()
}

func (a *UIPlugin) Manifest() jsondoc.JSONDoc {
	return jsondoc.New([]byte(a.Get("manifest").Raw))
}

func (a *UIPlugin) ActiveVersionID() string {
	return a.Get("activeVersionId").String()
}

func (a *UIPlugin) Owner() jsondoc.JSONDoc {
	return jsondoc.New([]byte(a.Get("owner").Raw))
}
