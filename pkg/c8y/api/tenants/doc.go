// Package tenants covers tenants (for the management tenant and enterprise
// tenants): create, read, update and delete tenants, their application
// subscriptions and two factor settings. Services for the current tenant are
// nested: Current, Options, SystemOptions, UsageStatistics, DeviceStatistics,
// RestrictedRoles and LoginTokens.
//
// Reach it from a client with client.Tenants.
//
// Example, with a client and ctx in scope:
//
//	for tenant, err := range client.Tenants.ListAll(ctx, tenants.ListOptions{}).Items() {
//		if err != nil {
//			break
//		}
//		fmt.Println(tenant.Get("id").String())
//	}
package tenants
