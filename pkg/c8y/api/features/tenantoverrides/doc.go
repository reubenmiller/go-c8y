// Package tenantoverrides provides enable, disable or remove a feature toggle
// for the current tenant or for a given tenant.
//
// Reach it from a client with client.Features.Tenants.
//
// Example, with a client and ctx in scope:
//
//	result := client.Features.Tenants.Enable(ctx, "my-feature")
//	fmt.Println(result.Err)
package tenantoverrides
