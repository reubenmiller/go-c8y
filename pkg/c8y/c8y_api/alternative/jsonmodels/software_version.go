package jsonmodels

import (
	"encoding/json"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
)

type SoftwareVersion struct {
	jsondoc.Facade
}

func NewSoftwareVersion(b []byte) SoftwareVersion {
	return SoftwareVersion{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func NewSoftwareVersionWithOptions(version, url string) SoftwareVersion {
	data := map[string]any{
		"type": "c8y_SoftwareBinary",
		"c8y_Software": map[string]any{
			"version": version,
		},
	}
	if url != "" {
		data["c8y_Software"].(map[string]any)["url"] = url
	}
	b, _ := json.Marshal(data)
	return SoftwareVersion{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (sv SoftwareVersion) ID() string {
	return sv.Get("id").String()
}

func (sv SoftwareVersion) Type() string {
	return sv.Get("type").String()
}

func (sv SoftwareVersion) Version() string {
	return sv.Get("c8y_Software.version").String()
}

func (sv SoftwareVersion) URL() string {
	return sv.Get("c8y_Software.url").String()
}

func (sv SoftwareVersion) Owner() string {
	return sv.Get("owner").String()
}

func (sv SoftwareVersion) CreationTime() time.Time {
	return sv.Get("creationTime").Time()
}

func (sv SoftwareVersion) LastUpdated() time.Time {
	return sv.Get("lastUpdated").Time()
}

func (sv SoftwareVersion) SoftwareID() string {
	return sv.Get(`additionParents.references.#(managedObject.type=="c8y_Software").managedObject.id"`).String()
}

func (sv SoftwareVersion) SoftwareName() string {
	return sv.Get(`additionParents.references.#(managedObject.type=="c8y_Software").managedObject.name"`).String()
}

func (sv SoftwareVersion) SoftwareSelf() string {
	return sv.Get(`additionParents.references.#(managedObject.type=="c8y_Software").managedObject.self"`).String()
}
func (sv SoftwareVersion) SoftwareType() string {
	return sv.Get(`additionParents.references.#(managedObject.type=="c8y_Software").managedObject.type"`).String()
}
