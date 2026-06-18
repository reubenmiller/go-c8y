package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// RoleReference is one entry of a role reference collection as returned by the
// user/group role endpoints (and by a single role assignment): a reference with
// its own self link wrapping the referenced role:
//
//	{ "self": ".../roles/ROLE_X", "role": { "self": ..., "id": "ROLE_X", "name": "ROLE_X" } }
//
// ID and Name delegate to the embedded role (a role's id equals its name in
// Cumulocity), so a RoleReference is interchangeable with a Role for the common
// "which role" question while still exposing the reference's own self link.
type RoleReference struct {
	jsondoc.Facade
}

func NewRoleReference(b []byte) RoleReference {
	return RoleReference{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// Self returns the reference's own self link (the assignment URL used to
// unassign the role).
func (r RoleReference) Self() string {
	return r.Get("self").String()
}

// Role returns the referenced role.
func (r RoleReference) Role() Role {
	return NewRole([]byte(r.Get("role").Raw))
}

// ID returns the referenced role's id (which equals the role name).
func (r RoleReference) ID() string {
	return r.Get("role.id").String()
}

// Name returns the referenced role's name.
func (r RoleReference) Name() string {
	return r.Get("role.name").String()
}
