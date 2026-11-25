// Package remoteaccess provides remote access to devices through the Cloud
// Remote Access (SSH, VNC, telnet and passthrough). This package holds the
// typed REST API for the configurations of a device. The websocket proxy
// client is in pkg/remoteaccess.
//
// Reach it from a client with client.RemoteAccess.Configurations.
//
// Example, with a client and ctx in scope:
//
//	configs := client.RemoteAccess.Configurations.List(ctx, remoteaccessconfigs.ListOptions{ManagedObjectID: "12345"})
//	for config, err := range configs.Items() {
//		if err != nil {
//			break
//		}
//		fmt.Println(config.Get("name").String(), config.Get("protocol").String())
//	}
package remoteaccess
