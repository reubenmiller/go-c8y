// Package firmwarepatches provides the patches of a firmware version. A patch
// depends on a given firmware version.
//
// Reach it from a client with client.Repository.Firmware.Patches.
//
// Example, with a client and ctx in scope:
//
//	result := client.Repository.Firmware.Patches.List(ctx, firmwarepatches.ListOptions{})
//	fmt.Println(result.Err)
package firmwarepatches
