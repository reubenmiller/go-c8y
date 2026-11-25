// Package users provides the users which are members of a user group.
//
// Reach it from a client with client.UserGroups.Users.
//
// Example, with a client and ctx in scope:
//
//	members := client.UserGroups.Users.List(ctx, groupusers.ListOptions{})
//	fmt.Println(members.Err)
package users
