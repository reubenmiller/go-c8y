// Package source provides resolvers which turn a human readable reference such
// as name:my-device, ext:c8y_Serial:ABC or query:... into an id. The services
// use it for device, application and similar references.
//
// The services use it in their ResolveID methods, for example
// client.ManagedObjects.ResolveID.
//
// Example, with a client and ctx in scope:
//
//	fmt.Println(source.IsID("12345"), source.ParseRef("name:my-device"))
package source
