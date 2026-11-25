// Package systemoptions provides read-only system options of the platform.
//
// Reach it from a client with client.Tenants.SystemOptions.
//
// Example, with a client and ctx in scope:
//
//	for option, err := range client.Tenants.SystemOptions.ListAll(ctx, systemoptions.ListOptions{}).Items() {
//		if err != nil {
//			break
//		}
//		fmt.Println(option.Get("category").String(), option.Get("key").String())
//	}
package systemoptions
