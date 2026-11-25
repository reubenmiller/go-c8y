// Package devices covers devices: managed objects which are devices, with
// find, get-or-create, device registration (client.Devices.Registration) and
// certificate enrollment (client.Devices.Enrollment).
//
// Reach it from a client with client.Devices.
//
// Example, with a client and ctx in scope:
//
//	device := client.Devices.GetOrCreateByName(ctx, "my-device", "example_Device", map[string]any{})
//	fmt.Println(device.Data.ID(), device.Err)
package devices
