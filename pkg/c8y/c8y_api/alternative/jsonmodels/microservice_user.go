package jsonmodels

import "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"

type MicroserviceUser struct {
	jsondoc.JSONDoc
}

func NewMicroserviceUser(b []byte) MicroserviceUser {
	return MicroserviceUser{jsondoc.New(b)}
}

// Username returns the username of the microservice user
func (u MicroserviceUser) Username() string {
	return u.Get("name").String()
}

// Password returns the password of the microservice user
func (u MicroserviceUser) Password() string {
	return u.Get("password").String()
}

// Tenant returns the tenant of the microservice user
func (u MicroserviceUser) Tenant() string {
	return u.Get("tenant").String()
}

type MicroserviceSetting struct {
	jsondoc.JSONDoc
}

func NewMicroserviceSetting(b []byte) MicroserviceSetting {
	return MicroserviceSetting{jsondoc.New(b)}
}

// Key returns the key of the setting
func (s MicroserviceSetting) Key() string {
	return s.Get("key").String()
}

// DefaultValue returns the default value of the setting
func (s MicroserviceSetting) DefaultValue() string {
	return s.Get("defaultValue").String()
}

// Editable returns whether the setting is editable
func (s MicroserviceSetting) Editable() bool {
	return s.Get("editable").Bool()
}

// InheritFromOwner returns whether the setting inherits from owner
func (s MicroserviceSetting) InheritFromOwner() bool {
	return s.Get("inheritFromOwner").Bool()
}
