package child

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/tidwall/gjson"
)

// ResultProperty is the gjson path to the child managed objects inside a
// managedObjectReferenceCollection response ({"references":[{"managedObject":{…}}]}).
// All child-reference endpoints (childAdditions/childAssets/childDevices) share it.
const ResultProperty = "references.#.managedObject"

type ListOptions struct {
	Query             string `url:"query,omitempty"`
	WithParents       bool   `url:"withParents,omitempty"`
	WithChildren      bool   `url:"withChildren,omitempty"`
	WithChildrenCount bool   `url:"withChildrenCount,omitempty"`

	// Pagination options
	pagination.PaginationOptions
}

// NewReferencedManagedObject parses a single managedObjectReference response
// ({"self":…,"managedObject":{…}}) returned by GET …/{childType}/{child} and
// returns its nested managedObject (the child). It falls back to the whole body
// when no managedObject wrapper is present, so a plain managed-object response
// still parses.
func NewReferencedManagedObject(b []byte) jsonmodels.ManagedObject {
	if mo := gjson.GetBytes(b, "managedObject"); mo.Exists() {
		return jsonmodels.NewManagedObject([]byte(mo.Raw))
	}
	return jsonmodels.NewManagedObject(b)
}
