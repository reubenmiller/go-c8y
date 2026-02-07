package jsonmodels

import (
	"encoding/json"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
)

type Configuration struct {
	jsondoc.Facade
}

func NewConfiguration(b []byte) Configuration {
	return Configuration{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func NewConfigurationWithOptions(name, configurationType string) Configuration {
	data := map[string]any{
		"type": "c8y_ConfigurationDump",
	}
	if name != "" {
		data["name"] = name
	}
	if configurationType != "" {
		data["configurationType"] = configurationType
	}
	b, _ := json.Marshal(data)
	return Configuration{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (c Configuration) ID() string {
	return c.Get("id").String()
}

func (c Configuration) Name() string {
	return c.Get("name").String()
}

func (c Configuration) Type() string {
	return c.Get("type").String()
}

func (c Configuration) ConfigurationType() string {
	return c.Get("configurationType").String()
}

func (c Configuration) Description() string {
	return c.Get("description").String()
}

func (c Configuration) URL() string {
	return c.Get("url").String()
}

func (c Configuration) Self() string {
	return c.Get("self").String()
}

// DeviceType returns the device type filter
func (c Configuration) DeviceType() string {
	return c.Get("deviceType").String()
}

func (c Configuration) Owner() string {
	return c.Get("owner").String()
}

func (c Configuration) CreationTime() time.Time {
	return c.Get("creationTime").Time()
}

func (c Configuration) LastUpdated() time.Time {
	return c.Get("lastUpdated").Time()
}
