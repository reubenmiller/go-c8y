// Package configuration provides configuration snapshots and configuration
// files of the repository (c8y_ConfigurationDump and c8y_Configuration managed
// objects).
//
// Reach it from a client with client.Repository.Configuration.
//
// Example, with a client and ctx in scope:
//
//	result := client.Repository.Configuration.Create(ctx, configuration.CreateOptions{
//		Name:              "ntp-config",
//		ConfigurationType: "ntp.conf",
//		URL:               "https://example.com/ntp.conf",
//	})
//	fmt.Println(result.Err)
package configuration
