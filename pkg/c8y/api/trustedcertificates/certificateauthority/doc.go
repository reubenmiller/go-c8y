// Package certificateauthority provides the certificate authority of the
// tenant, which signs the certificates of enrolled devices.
//
// Reach it from a client with client.TrustedCertificates.CertificateAuthority.
//
// Example, with a client and ctx in scope:
//
//	result := client.TrustedCertificates.CertificateAuthority.Get(ctx, certificateauthority.GetOptions{})
//	fmt.Println(result.Err)
package certificateauthority
