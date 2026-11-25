// Package userroles provides read-only access to the global user roles of the
// tenant (for example ROLE_ALARM_READ), with helpers to build role references.
// Assigning roles to users and groups is under the Users and Groups sub-
// services.
//
// Reach it from a client with client.UserRoles.
//
// Example, with a client and ctx in scope:
//
//	roles := client.UserRoles.List(ctx, userroles.ListOptions{})
//	for role, err := range roles.Items() {
//		if err != nil {
//			break
//		}
//		fmt.Println(role.Get("id").String())
//	}
package userroles
