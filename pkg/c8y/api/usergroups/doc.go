// Package usergroups provides user groups of a tenant, referenced by id or by
// name with usergroups.ByID and usergroups.ByName. Members are under
// client.UserGroups.Users.
//
// Reach it from a client with client.UserGroups.
//
// Example, with a client and ctx in scope:
//
//	group := client.UserGroups.Get(ctx, usergroups.GetOptions{ID: usergroups.ByName("admins")})
//	fmt.Println(group.Data.Get("id").String(), group.Err)
package usergroups
