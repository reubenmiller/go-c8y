// Package firmwareversions provides the versions of a firmware item.
//
// Reach it from a client with client.Repository.Firmware.Versions.
//
// Example, with a client and ctx in scope:
//
//	result := client.Repository.Firmware.Versions.List(ctx, firmwareversions.ListOptions{})
//	fmt.Println(result.Err)
package firmwareversions
