// Package firmwareitems provides the firmware items of the repository, without
// their versions.
//
// Reach it from a client with client.Repository.Firmware.
//
// Example, with a client and ctx in scope:
//
//	result := client.Repository.Firmware.List(ctx, firmwareitems.ListOptions{})
//	fmt.Println(result.Err)
package firmwareitems
