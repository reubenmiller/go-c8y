// Package currenttenant provides information about the tenant the client is
// authenticated as, including its domain and subscribed applications.
//
// Reach it from a client with client.Tenants.Current.
//
// Example, with a client and ctx in scope:
//
//	result := client.Tenants.Current.Get(ctx, currenttenant.GetOptions{})
//	fmt.Println(result.Data.Get("name").String(), result.Err)
package currenttenant
