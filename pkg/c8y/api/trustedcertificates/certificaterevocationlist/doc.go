// Package certificaterevocationlist provides the certificate revocation list
// of the tenant.
//
// Reach it from a client with client.TrustedCertificates.RevocationList.
//
// Example, with a client and ctx in scope:
//
//	result := client.TrustedCertificates.RevocationList.List(ctx, revocation.GetOptions{})
//	fmt.Println(result.Err)
package certificaterevocationlist
