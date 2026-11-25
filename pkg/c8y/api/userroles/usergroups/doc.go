// Package usergroups lets you assign and unassign global roles of a user
// group.
//
// Reach it from a client with client.UserRoles.Groups.
//
// Example, with a client and ctx in scope:
//
//	result := client.UserRoles.Groups.List(ctx, userrolesgroups.ListOptions{})
//	fmt.Println(result.Err)
package usergroups
