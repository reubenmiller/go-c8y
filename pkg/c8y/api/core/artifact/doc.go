// Package artifact provides parsing of artifact file names such as software
// and firmware package names.
//
// It is used by the repository services.
//
// Example, with a client and ctx in scope:
//
//	fmt.Println(artifact.ParseName("my-agent_1.0.0_arm64.deb"))
package artifact
