package jsonmodels

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"

type ManagedObjectUser struct {
	jsondoc.Facade
}

func NewManagedObjectUser(b []byte) ManagedObjectUser {
	return ManagedObjectUser{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// UserName returns the username of the device's owner.
func (m ManagedObjectUser) UserName() string {
	return m.Get("userName").String()
}

// Enabled reports whether the device owner's account is enabled.
func (m ManagedObjectUser) Enabled() bool {
	return m.Get("enabled").Bool()
}

// Self returns the self link of this resource.
func (m ManagedObjectUser) Self() string {
	return m.Get("self").String()
}
