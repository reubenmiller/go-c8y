// Package software provides software items of the repository, and their
// versions (client.Repository.Software.Versions). CreateVersion finds or
// creates the item by name and uploads the binary in one call.
//
// Reach it from a client with client.Repository.Software.
//
// Example, with a client and ctx in scope:
//
//	version := client.Repository.Software.Versions.CreateVersion(ctx, softwareversions.CreateVersionOptions{
//		SoftwareName: "my-agent",
//		SoftwareType: "apt",
//		Version:      "1.0.0",
//		URL:          "https://example.com/my-agent_1.0.0.deb",
//	})
//	fmt.Println(version.Err)
package software
