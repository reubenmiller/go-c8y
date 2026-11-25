// Package identity covers external identities: map the id of a device in
// another system (serial number, MAC address) to its managed object. For
// lookups by external id prefer client.ManagedObjects.ResolveID with an "ext:"
// reference.
//
// Reach it from a client with client.Identity.
//
// Example, with a client and ctx in scope:
//
//	result := client.Identity.Get(ctx, identity.IdentityOptions{Type: "c8y_Serial", ExternalID: "ABC-123"})
//	if api.IsNotFound(result.Err) {
//		fmt.Println("not registered")
//	}
package identity
