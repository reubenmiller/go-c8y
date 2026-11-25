// Package softwareversions provides the versions of a software item, each with
// a binary or a URL.
//
// Reach it from a client with client.Repository.Software.Versions.
//
// Example, with a client and ctx in scope:
//
//	result := client.Repository.Software.Versions.List(ctx, softwareversions.ListOptions{})
//	for version, err := range result.Items() {
//		if err != nil {
//			break
//		}
//		fmt.Println(version.Get("c8y_Software.version").String())
//	}
package softwareversions
