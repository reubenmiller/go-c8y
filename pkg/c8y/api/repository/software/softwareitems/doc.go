// Package softwareitems provides the software items of the repository
// (c8y_Software managed objects), without their versions.
//
// Reach it from a client with client.Repository.Software.
//
// Example, with a client and ctx in scope:
//
//	result := client.Repository.Software.GetOrCreateByName(ctx, "my-agent", "apt", map[string]any{})
//	fmt.Println(result.Data.Get("id").String(), result.Err)
package softwareitems
