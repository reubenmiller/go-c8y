// Package users provides users of a tenant, referenced by name, and the
// current user (client.Users.Current) and device permissions
// (client.Users.DevicePermissions). ResolveID accepts a user name with *
// wildcards.
//
// Reach it from a client with client.Users.
//
// Example, with a client and ctx in scope:
//
//	id, err := client.Users.ResolveID(ctx, "alice*", nil)
//	fmt.Println(id, err)
package users
