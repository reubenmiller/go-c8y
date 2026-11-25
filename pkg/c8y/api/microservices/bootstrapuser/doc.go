// Package bootstrapuser lets you get the bootstrap user of a microservice,
// which a microservice uses to read its subscriptions.
//
// Reach it from a client with client.Microservices.BootstrapUser.
//
// Example, with a client and ctx in scope:
//
//	result := client.Microservices.BootstrapUser.Get(ctx, "12345")
//	fmt.Println(result.Data.Get("username").String(), result.Err)
package bootstrapuser
