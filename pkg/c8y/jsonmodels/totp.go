package jsonmodels

import "github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"

// TOTPSecret represents the response from the TOTP secret generation endpoint.
type TOTPSecret struct {
	jsondoc.JSONDoc
}

func NewTOTPSecret(b []byte) TOTPSecret {
	return TOTPSecret{jsondoc.New(b)}
}

// SecretQRURL returns the URL of the QR code image for scanning with an authenticator app.
func (t TOTPSecret) SecretQRURL() string {
	return t.Get("secretQrUrl").String()
}

// RawSecret returns the raw TOTP secret string for manual entry into an authenticator app.
func (t TOTPSecret) RawSecret() string {
	return t.Get("rawSecret").String()
}
