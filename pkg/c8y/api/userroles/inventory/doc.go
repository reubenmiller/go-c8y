// Package inventoryroles covers inventory roles: named sets of permissions on
// managed objects which can be assigned to users for a device or group.
//
// Reach it from a client with client.UserRoles.Inventory.
//
// Example, with a client and ctx in scope:
//
//	for role, err := range client.UserRoles.Inventory.ListAll(ctx, inventoryroles.ListOptions{}).Items() {
//		if err != nil {
//			break
//		}
//		fmt.Println(role.Get("name").String())
//	}
package inventoryroles
