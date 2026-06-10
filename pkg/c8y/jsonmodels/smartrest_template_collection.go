package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// SmartRestTemplateCollection is a SmartREST 2.0 template collection stored
// as an inventory managed object (type c8y_SmartRest2Template). The
// collection name doubles as the external identity
// (c8y_SmartRest2DeviceIdentifier) which devices reference as the X-Id.
type SmartRestTemplateCollection struct {
	jsondoc.Facade
}

func NewSmartRestTemplateCollection(b []byte) SmartRestTemplateCollection {
	return SmartRestTemplateCollection{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (c SmartRestTemplateCollection) ID() string {
	return c.Get("id").String()
}

func (c SmartRestTemplateCollection) Name() string {
	return c.Get("name").String()
}

func (c SmartRestTemplateCollection) Self() string {
	return c.Get("self").String()
}

// ExternalID returns the __externalId fragment the platform stores on
// exported template collections
func (c SmartRestTemplateCollection) ExternalID() string {
	return c.Get("__externalId").String()
}
