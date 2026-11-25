// Package loginoptions provides login options of the tenant (basic, OAuth2
// internal, OAuth2 SSO, and others), including the access mappings under
// client.LoginOptions.AccessMappings.
//
// Reach it from a client with client.LoginOptions.
//
// Example, with a client and ctx in scope:
//
//	for option, err := range client.LoginOptions.ListAll(ctx, loginoptions.ListOptions{}).Items() {
//		if err != nil {
//			break
//		}
//		fmt.Println(option.Get("type").String())
//	}
package loginoptions
