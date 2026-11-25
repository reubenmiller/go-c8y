// Package users lets you assign and unassign global roles of a user.
//
// Reach it from a client with client.UserRoles.Users.
//
// Example, with a client and ctx in scope:
//
//	result := client.UserRoles.Users.List(ctx, userrolesusers.ListOptions{})
//	fmt.Println(result.Err)
package users
