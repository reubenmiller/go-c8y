// Package currentapplication covers the application the client is
// authenticated as: read its subscriptions and settings.
//
// Reach it from a client with client.Applications.Current.
//
// Example, with a client and ctx in scope:
//
//	result := client.Applications.Current.Get(ctx)
//	fmt.Println(result.Err)
package currentapplication
