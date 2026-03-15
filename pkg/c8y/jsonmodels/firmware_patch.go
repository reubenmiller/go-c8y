package jsonmodels

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"

// FirmwarePatch represents a firmware patch in Cumulocity
type FirmwarePatch struct {
	jsondoc.Facade
}

// NewFirmwarePatch creates a FirmwarePatch from raw bytes
func NewFirmwarePatch(data []byte) FirmwarePatch {
	return FirmwarePatch{jsondoc.Facade{JSONDoc: jsondoc.New(data)}}
}

// ID returns the patch's ID
func (f FirmwarePatch) ID() string {
	return f.Get("id").String()
}

// Name returns the patch's name
func (f FirmwarePatch) Name() string {
	return f.Get("name").String()
}

// Type returns the patch's type (should be "c8y_FirmwareBinary")
func (f FirmwarePatch) Type() string {
	return f.Get("type").String()
}

// Version returns the patch's version from c8y_Firmware.version
func (f FirmwarePatch) Version() string {
	return f.Get("c8y_Firmware.version").String()
}

// URL returns the patch's URL from c8y_Firmware.url
func (f FirmwarePatch) URL() string {
	return f.Get("c8y_Firmware.url").String()
}

// Dependency returns the dependency version from c8y_Patch.dependency
func (f FirmwarePatch) Dependency() string {
	return f.Get("c8y_Patch.dependency").String()
}

// FirmwareID returns the parent firmware ID from additionParents
func (f FirmwarePatch) FirmwareID() string {
	return f.Get(`additionParents.references.#(managedObject.type=="c8y_Firmware").managedObject.id`).String()
}
