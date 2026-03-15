package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// DevicePermissionOwners represents the response from
//
//	GET /user/devicePermissions/{id}
//
// where {id} is a managed-object ID. The platform returns the list of users
// and groups that have device-level permissions on that managed object.
//
// OAS schema: DevicePermissionOwners
//
// Example shape:
//
//	{
//	  "users": [
//	    {
//	      "id": "jdoe",
//	      "userName": "jdoe",
//	      "devicePermissions": {
//	        "12345": ["MANAGED_OBJECT:*:ADMIN"]
//	      }
//	    }
//	  ],
//	  "groups": [
//	    {
//	      "id": 7,
//	      "devicePermissions": {
//	        "12345": ["READ"]
//	      }
//	    }
//	  ]
//	}
type DevicePermissionOwners struct {
	jsondoc.Facade
}

// NewDevicePermissionOwners parses raw JSON bytes into a DevicePermissionOwners value.
func NewDevicePermissionOwners(b []byte) DevicePermissionOwners {
	return DevicePermissionOwners{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// UserNames returns the username of each user entry in the response.
func (d DevicePermissionOwners) UserNames() []string {
	var names []string
	for _, u := range d.Get("users").Array() {
		if name := u.Get("userName").String(); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// GroupIDs returns the numeric ID of each group entry in the response.
func (d DevicePermissionOwners) GroupIDs() []int64 {
	var ids []int64
	for _, g := range d.Get("groups").Array() {
		ids = append(ids, g.Get("id").Int())
	}
	return ids
}

// InventoryRoleAssignment represents the assignment of one or more inventory
// roles to a user for a specific managed object.
//
// OAS schema: inventoryAssignment
//
// Example shape returned by the platform:
//
//	{
//	  "id": 1,
//	  "self": "https://.../user/{tenantId}/users/{userId}/roles/inventory/1",
//	  "managedObject": "12345",
//	  "roles": [
//	    { "id": 4, "name": "Operations: Restart Device", "description": "Can restart devices.", "self": "..." }
//	  ]
//	}
type InventoryRoleAssignment struct {
	jsondoc.Facade
}

// NewInventoryRoleAssignment parses raw JSON bytes into an InventoryRoleAssignment.
func NewInventoryRoleAssignment(b []byte) InventoryRoleAssignment {
	return InventoryRoleAssignment{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// ID returns the numeric identifier of this assignment.
func (a InventoryRoleAssignment) ID() int64 {
	return a.Get("id").Int()
}

// Self returns the self URI link for this assignment.
func (a InventoryRoleAssignment) Self() string {
	return a.Get("self").String()
}

// ManagedObjectID returns the ID of the managed object this assignment
// targets. Per the OAS spec (inventoryAssignment.managedObject), this is a
// plain string containing the managed-object ID.
func (a InventoryRoleAssignment) ManagedObjectID() string {
	return a.Get("managedObject").String()
}

// RoleIDs returns the numeric IDs of all inventory roles included in this
// assignment.
func (a InventoryRoleAssignment) RoleIDs() []int64 {
	var ids []int64
	for _, v := range a.Get("roles.#.id").Array() {
		ids = append(ids, v.Int())
	}
	return ids
}
