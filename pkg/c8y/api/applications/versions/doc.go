// Package versions provides the versions of an application, with tags such as
// latest.
//
// Reach it from a client with client.Applications.Versions
// (client.ApplicationVersions is deprecated).
//
// Example, with a client and ctx in scope:
//
//	result := client.Applications.Versions.ListByTag(ctx, "12345", "latest")
//	fmt.Println(result.Err)
package versions
