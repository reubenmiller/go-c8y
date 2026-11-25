// Package devicepermissions provides permissions of users on devices, and the
// inventory role assignments of a user.
//
// Reach it from a client with client.Users.DevicePermissions.
//
// Example, with a client and ctx in scope:
//
//	result := client.Users.DevicePermissions.GetDevicePermissions(ctx, devicepermissions.GetDevicePermissionsOptions{})
//	fmt.Println(result.Err)
package devicepermissions
