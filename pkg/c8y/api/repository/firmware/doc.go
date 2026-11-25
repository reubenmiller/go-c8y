// Package firmware provides firmware items of the repository, and their
// versions and patches (client.Repository.Firmware.Versions and .Patches).
//
// Reach it from a client with client.Repository.Firmware.
//
// Example, with a client and ctx in scope:
//
//	version := client.Repository.Firmware.Versions.CreateVersion(ctx, firmwareversions.CreateVersionOptions{
//		FirmwareName: "my-firmware",
//		Version:      "2.0.0",
//		URL:          "https://example.com/firmware-2.0.0.img",
//	})
//	fmt.Println(version.Err)
package firmware
