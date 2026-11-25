// Package currentmicroservice provides the microservice the client is
// authenticated as. Only works with microservice credentials.
//
// Reach it from a client with client.Microservices.Current.
//
// Example, with a client and ctx in scope:
//
//	result := client.Microservices.Current.ListSettings(ctx)
//	fmt.Println(result.Err)
package currentmicroservice
