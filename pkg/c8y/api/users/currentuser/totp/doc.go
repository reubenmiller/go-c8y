// Package totp provides tOTP (authenticator app) two factor settings of the
// current user.
//
// Reach it from a client with client.Users.Current.TOTP.
//
// Example, with a client and ctx in scope:
//
//	result := client.Users.Current.TOTP.GetActivity(ctx)
//	fmt.Println(result.Err)
package totp
