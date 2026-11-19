package jsonmodels

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"

// CertificateAccessToken is the response of POST /tenant/oauth/certificate.
// AccessToken is a compact JWE encrypted to the certificate's public key; it
// must be decrypted with the private key before use.
type CertificateAccessToken struct {
	jsondoc.JSONDoc
}

func NewCertificateAccessToken(b []byte) CertificateAccessToken {
	return CertificateAccessToken{jsondoc.New(b)}
}

// AccessToken returns the encrypted access token (compact JWE).
func (t CertificateAccessToken) AccessToken() string {
	return t.Get("access_token").String()
}

// TokenType returns the token type, usually "Bearer".
func (t CertificateAccessToken) TokenType() string {
	return t.Get("token_type").String()
}

// ExpiresIn returns the token lifetime in seconds, or 0 when not given.
func (t CertificateAccessToken) ExpiresIn() int64 {
	return t.Get("expires_in").Int()
}
