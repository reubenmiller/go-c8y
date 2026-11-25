// Package registration covers device registration: new device requests, their
// credentials, bulk registration and the combined Register call.
//
// Reach it from a client with client.Devices.Registration.
//
// Example, with a client and ctx in scope:
//
//	result := client.Devices.Registration.Register(ctx, registration.RegisterOptions{
//		ExternalID: "ABC-123",
//		Name:       "my-device",
//	})
//	fmt.Println(result.Err)
package registration
