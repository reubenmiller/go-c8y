// Package events covers events: create, query, update and delete events, with
// binaries attached to an event (client.Events.Binaries), and follow events in
// realtime with SubscribeStream.
//
// Reach it from a client with client.Events.
//
// Example, with a client and ctx in scope:
//
//	created := client.Events.Create(ctx, events.CreateOptions{
//		Source: managedobjects.ByName("my-device"),
//		Type:   "example_DoorOpened",
//		Text:   "Door opened",
//	})
//	fmt.Println(created.Data.ID(), created.Err)
package events
