// Package remoteaccess_configurations provides the remote access
// configurations of a device (name, protocol, hostname, port) and detection of
// an existing one.
//
// Reach it from a client with client.RemoteAccess.Configurations.
//
// Example, with a client and ctx in scope:
//
//	id, err := client.RemoteAccess.Configurations.ResolveID(ctx, "12345", "ssh")
//	fmt.Println(id, err)
package remoteaccess_configurations
