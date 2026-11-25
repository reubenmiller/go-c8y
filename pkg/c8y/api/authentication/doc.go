// Package authentication covers credentials and token sources: AuthOptions
// (basic, token, certificate, OAuth2) read from the environment or a lookup
// function, token parsing and caching, and per call credentials with WithAuth.
//
// Pass the credentials in the Auth field of api.ClientOptions.
//
// Example, with a client and ctx in scope:
//
//	auth := authentication.AuthOptions{
//		Tenant:   "t12345",
//		Username: "user",
//		Password: "password",
//		AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
//	}
//	fmt.Println(auth.Tenant)
package authentication
