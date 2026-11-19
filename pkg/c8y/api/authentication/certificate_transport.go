package authentication

import (
	"crypto/rsa"
	"fmt"
)

// CertificateTransport selects how a client certificate is exchanged for an
// access token.
type CertificateTransport string

const (
	// CertificateTransportAuto (the default) chooses by key type: RSA keys use
	// the HTTPS exchange, all other keys use mTLS. It never falls back from one
	// transport to the other.
	CertificateTransportAuto CertificateTransport = ""

	// CertificateTransportHTTPS uses POST /tenant/oauth/certificate on the
	// standard HTTPS port. Only RSA keys are supported: the platform returns
	// the token encrypted to the certificate's public key.
	CertificateTransportHTTPS CertificateTransport = "HTTPS"

	// CertificateTransportMTLS uses POST /devicecontrol/deviceAccessToken over
	// mutual TLS on the client's MTLSPort (default 8443).
	CertificateTransportMTLS CertificateTransport = "MTLS"
)

// ResolveCertificateTransport returns the transport to use for the given
// private key. It returns an error when HTTPS is forced with a non-RSA key.
func ResolveCertificateTransport(t CertificateTransport, privateKey any) (CertificateTransport, error) {
	_, isRSA := privateKey.(*rsa.PrivateKey)
	switch t {
	case CertificateTransportAuto:
		if isRSA {
			return CertificateTransportHTTPS, nil
		}
		return CertificateTransportMTLS, nil
	case CertificateTransportHTTPS:
		if !isRSA {
			return "", fmt.Errorf("certificate login over HTTPS needs an RSA key, got %T: use CertificateTransportMTLS (port 8443) for EC and Ed25519 keys", privateKey)
		}
		return CertificateTransportHTTPS, nil
	case CertificateTransportMTLS:
		return CertificateTransportMTLS, nil
	default:
		return "", fmt.Errorf("unknown certificate transport %q", t)
	}
}
