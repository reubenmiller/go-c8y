package jsonmodels

import (
	"encoding/json"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsondoc"
)

type FirmwareVersion struct {
	jsondoc.Facade
}

func NewFirmwareVersion(b []byte) FirmwareVersion {
	return FirmwareVersion{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func NewFirmwareVersionWithOptions(version, url string) FirmwareVersion {
	data := map[string]any{
		"type": "c8y_FirmwareBinary",
		"c8y_Firmware": map[string]any{
			"version": version,
		},
	}
	if url != "" {
		data["c8y_Firmware"].(map[string]any)["url"] = url
	}
	b, _ := json.Marshal(data)
	return FirmwareVersion{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (fv FirmwareVersion) ID() string {
	return fv.Get("id").String()
}

func (fv FirmwareVersion) Type() string {
	return fv.Get("type").String()
}

func (fv FirmwareVersion) Version() string {
	return fv.Get("c8y_Firmware.version").String()
}

func (fv FirmwareVersion) URL() string {
	return fv.Get("c8y_Firmware.url").String()
}

func (fv FirmwareVersion) Owner() string {
	return fv.Get("owner").String()
}

func (fv FirmwareVersion) CreationTime() time.Time {
	return fv.Get("creationTime").Time()
}

func (fv FirmwareVersion) LastUpdated() time.Time {
	return fv.Get("lastUpdated").Time()
}

func (fv FirmwareVersion) FirmwareID() string {
	return fv.Get(`additionParents.references.#(managedObject.type=="c8y_Firmware").managedObject.id`).String()
}

func (fv FirmwareVersion) FirmwareName() string {
	return fv.Get(`additionParents.references.#(managedObject.type=="c8y_Firmware").managedObject.name`).String()
}

func (fv FirmwareVersion) FirmwareSelf() string {
	return fv.Get(`additionParents.references.#(managedObject.type=="c8y_Firmware").managedObject.self`).String()
}

func (fv FirmwareVersion) FirmwareType() string {
	return fv.Get(`additionParents.references.#(managedObject.type=="c8y_Firmware").managedObject.type`).String()
}
