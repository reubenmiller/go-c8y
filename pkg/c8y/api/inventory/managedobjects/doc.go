// Package managedobjects covers the inventory: managed objects (devices,
// groups, assets and any other object) with typed references (ByID, ByName,
// ByExternalID, ByQuery) and the ResolveID resolver, get-or-create helpers,
// bulk create and update, and child references under ChildDevices, ChildAssets
// and ChildAdditions.
//
// Reach it from a client with client.ManagedObjects.
//
// Example, with a client and ctx in scope:
//
//	for mo, err := range client.ManagedObjects.ListAll(ctx, managedobjects.ListOptions{Type: "example_Sensor"}).Items() {
//		if err != nil {
//			break
//		}
//		fmt.Println(mo.Get("id").String(), mo.Get("name").String())
//	}
package managedobjects
