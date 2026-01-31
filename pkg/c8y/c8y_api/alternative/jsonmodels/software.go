package jsonmodels

import (
	"encoding/json"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
)

type Software struct {
	jsondoc.Facade
}

func NewSoftware(b []byte) Software {
	return Software{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func NewSoftwareWithOptions(name, softwareType string) Software {
	data := map[string]any{
		"type": "c8y_Software",
	}
	if name != "" {
		data["name"] = name
	}
	if softwareType != "" {
		data["softwareType"] = softwareType
	}
	b, _ := json.Marshal(data)
	return Software{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (s Software) ID() string {
	return s.Get("id").String()
}

func (s Software) Name() string {
	return s.Get("name").String()
}

func (s Software) Type() string {
	return s.Get("type").String()
}

func (s Software) SoftwareType() string {
	return s.Get("softwareType").String()
}

func (s Software) Description() string {
	return s.Get("description").String()
}

func (s Software) Owner() string {
	return s.Get("owner").String()
}

func (s Software) CreationTime() time.Time {
	return s.Get("creationTime").Time()
}

func (s Software) LastUpdated() time.Time {
	return s.Get("lastUpdated").Time()
}
