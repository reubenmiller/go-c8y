// Package currentuser covers the user the client is authenticated as: read and
// update the profile and password, and the TOTP settings
// (client.Users.Current.TOTP).
//
// Reach it from a client with client.Users.Current (client.Users.CurrentUser
// is deprecated).
//
// Example, with a client and ctx in scope:
//
//	me := client.Users.Current.Get(ctx)
//	fmt.Println(me.Data.Get("userName").String(), me.Err)
package currentuser
