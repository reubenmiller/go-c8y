// Package tenantoptions covers tenant options: key value settings grouped by
// category.
//
// Reach it from a client with client.Tenants.Options.
//
// Example, with a client and ctx in scope:
//
//	result := client.Tenants.Options.Get(ctx, tenantoptions.GetOption{Category: "access.control", Key: "allow.origin"})
//	fmt.Println(result.Data.Get("value").String(), result.Err)
package tenantoptions
