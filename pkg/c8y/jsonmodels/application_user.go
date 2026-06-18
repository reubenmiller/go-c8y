package jsonmodels

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"

// ApplicationUser is a subscribed user of the current application, as returned by
// the applicationUserCollection response of /application/currentApplication/subscriptions.
type ApplicationUser struct {
	jsondoc.JSONDoc
}

func NewApplicationUser(b []byte) ApplicationUser {
	return ApplicationUser{jsondoc.New(b)}
}

// Username returns the username of the application subscription user
func (u ApplicationUser) Username() string {
	return u.Get("name").String()
}

// Password returns the password of the application subscription user
func (u ApplicationUser) Password() string {
	return u.Get("password").String()
}

// Tenant returns the tenant of the application subscription user
func (u ApplicationUser) Tenant() string {
	return u.Get("tenant").String()
}
