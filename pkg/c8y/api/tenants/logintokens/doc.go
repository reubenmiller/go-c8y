// Package logintokens covers login tokens: exchange credentials or a
// certificate for an OAI-Secure token or cookies.
//
// Reach it from a client with client.Tenants.LoginTokens (client.LoginTokens
// is deprecated).
//
// Example, with a client and ctx in scope:
//
//	result := client.Tenants.LoginTokens.Create(ctx, logintokens.CreateTokenOptions{})
//	fmt.Println(result.Err)
package logintokens
