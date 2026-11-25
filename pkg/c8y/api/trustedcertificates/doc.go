// Package trustedcertificates provides trusted certificates used for
// certificate based device authentication, with the certificate authority
// (client.TrustedCertificates.CertificateAuthority) and the revocation list
// (client.TrustedCertificates.RevocationList).
//
// Reach it from a client with client.TrustedCertificates.
//
// Example, with a client and ctx in scope:
//
//	result := client.TrustedCertificates.Get(ctx, trustedcertificates.GetOptions{Fingerprint: "ab12cd34"})
//	fmt.Println(result.Data.Get("name").String(), result.Err)
package trustedcertificates
