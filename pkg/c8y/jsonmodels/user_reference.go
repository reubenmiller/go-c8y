package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// UserReference is one entry of a user reference collection as returned by the
// user-group membership endpoints (and by a single add-user-to-group call): a
// reference with its own self link wrapping the referenced user:
//
//	{ "self": ".../groups/42/users/peterpi%40example.com", "user": { "self": ..., "id": "peterpi@example.com", "userName": "peterpi@example.com" } }
//
// ID and Name delegate to the embedded user (a user's id equals its userName in
// Cumulocity), so a UserReference is interchangeable with a User for the common
// "which user" question while still exposing the reference's own self link.
type UserReference struct {
	jsondoc.Facade
}

func NewUserReference(b []byte) UserReference {
	return UserReference{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// Self returns the reference's own self link (the membership URL used to remove
// the user from the group).
func (r UserReference) Self() string {
	return r.Get("self").String()
}

// User returns the referenced user.
func (r UserReference) User() User {
	return NewUser([]byte(r.Get("user").Raw))
}

// ID returns the referenced user's id (which equals the user name).
func (r UserReference) ID() string {
	return r.Get("user.id").String()
}

// Name returns the referenced user's name (its userName).
func (r UserReference) Name() string {
	return r.Get("user.userName").String()
}
