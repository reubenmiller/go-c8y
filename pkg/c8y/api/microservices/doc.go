// Package microservices covers microservice applications: deploy a
// microservice zip, read its status, subscribe tenants, and helpers for the
// manifest and the bootstrap or service user environment variables.
//
// Reach it from a client with client.Microservices.
//
// Example, with a client and ctx in scope:
//
//	result := client.Microservices.Get(ctx, client.Microservices.ByName("my-service"))
//	if api.IsNotFound(result.Err) {
//		fmt.Println("not deployed")
//	}
package microservices
