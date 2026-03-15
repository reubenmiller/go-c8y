package jsonmodels

import (
	"encoding/json"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

type Firmware struct {
	jsondoc.Facade
}

func NewFirmware(b []byte) Firmware {
	return Firmware{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func NewFirmwareWithOptions(name string) Firmware {
	data := map[string]any{
		"type": "c8y_Firmware",
	}
	if name != "" {
		data["name"] = name
	}
	b, _ := json.Marshal(data)
	return Firmware{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (f Firmware) ID() string {
	return f.Get("id").String()
}

func (f Firmware) Name() string {
	return f.Get("name").String()
}

func (f Firmware) Type() string {
	return f.Get("type").String()
}

func (f Firmware) Description() string {
	return f.Get("description").String()
}

func (f Firmware) URL() string {
	return f.Get("url").String()
}

func (f Firmware) Self() string {
	return f.Get("self").String()
}

// DeviceType returns the device type filter from c8y_Filter.type
func (f Firmware) DeviceType() string {
	return f.Get("c8y_Filter.type").String()
}

func (f Firmware) Owner() string {
	return f.Get("owner").String()
}

func (f Firmware) CreationTime() time.Time {
	return f.Get("creationTime").Time()
}

func (f Firmware) LastUpdated() time.Time {
	return f.Get("lastUpdated").Time()
}
