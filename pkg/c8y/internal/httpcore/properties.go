package httpcore

// Result properties shared by services: the gjson path of the items in a
// collection response of an endpoint which more than one service uses.
const (
	// ResultPropertyManagedObjects is the items of the inventory collection,
	// also used by the services which are views on managed objects (devices,
	// agents, groups, ...).
	ResultPropertyManagedObjects = "managedObjects"

	// ResultPropertyApplications is the items of the applications collection.
	ResultPropertyApplications = "applications"

	// ResultPropertyChildReferences is the child managed objects inside a
	// child references response.
	ResultPropertyChildReferences = "references.#.managedObject"
)
