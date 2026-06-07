package jsonmodels

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"

// AccessMapping represents a login-option access mapping (mapping a condition to a set
// of applications and groups granted on SSO login). The structured `when`/`thenGroups`/
// `thenApplications` fields are accessed via the Facade's Get(); the common scalars have
// typed accessors.
type AccessMapping struct {
	jsondoc.Facade
}

func NewAccessMapping(b []byte) AccessMapping {
	return AccessMapping{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// ID returns the unique identifier of the access mapping.
func (a AccessMapping) ID() string {
	return a.Get("id").String()
}
