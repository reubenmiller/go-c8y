package jsonmodels

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"

// InventoryAccessMapping represents a login-option inventory access mapping (mapping a
// condition to a set of inventory-role assignments granted on SSO login). The structured
// `when`/`thenInventoryRoles` fields are accessed via the Facade's Get(); the common
// scalars have typed accessors.
type InventoryAccessMapping struct {
	jsondoc.Facade
}

func NewInventoryAccessMapping(b []byte) InventoryAccessMapping {
	return InventoryAccessMapping{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// ID returns the unique identifier of the inventory access mapping.
func (a InventoryAccessMapping) ID() string {
	return a.Get("id").String()
}
