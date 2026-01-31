package jsonmodels

import "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"

type BootstrapUser struct {
	jsondoc.JSONDoc
}

func NewBootstrapUser(b []byte) BootstrapUser {
	return BootstrapUser{jsondoc.New(b)}
}

// Username returns the username of the bootstrap user
func (u BootstrapUser) Username() string {
	return u.Get("name").String()
}

// Password returns the password of the bootstrap user
func (u BootstrapUser) Password() string {
	return u.Get("password").String()
}

// Tenant returns the tenant of the bootstrap user
func (u BootstrapUser) Tenant() string {
	return u.Get("tenant").String()
}
