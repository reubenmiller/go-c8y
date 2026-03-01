package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
)

// InventoryRole represents a Cumulocity inventory role (scoped permission set).
type InventoryRole struct {
	jsondoc.Facade
}

// NewInventoryRole parses raw JSON bytes into an InventoryRole.
func NewInventoryRole(b []byte) InventoryRole {
	return InventoryRole{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// ID returns the numeric identifier of the inventory role.
func (r InventoryRole) ID() int64 {
	return r.Get("id").Int()
}

// Name returns the name of the inventory role.
func (r InventoryRole) Name() string {
	return r.Get("name").String()
}

// Description returns the human-readable description of the inventory role.
func (r InventoryRole) Description() string {
	return r.Get("description").String()
}

// Self returns the self URI link for the inventory role.
func (r InventoryRole) Self() string {
	return r.Get("self").String()
}
